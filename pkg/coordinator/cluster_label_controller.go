package coordinator

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/tools"

	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cluster-manager/pkg/client/listers/containership.io/v3"
)

const (
	clusterLabelControllerName = "ClusterLabelController"

	clusterLabelDelayBetweenRetries = 30 * time.Second

	maxClusterLabelControllerRetries = 10
)

// ClusterLabelController is the controller implementation for the containership
// upgrading clusters to a users desired kubernetes version
type ClusterLabelController struct {
	kubeclientset kubernetes.Interface
	csclientset   csclientset.Interface

	clusterLabelLister  cslisters.ClusterLabelLister
	clusterLabelsSynced cache.InformerSynced
	nodeLister          corelistersv1.NodeLister
	nodesSynced         cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewClusterLabelController returns a new clusterLabel controller
func NewClusterLabelController(kubeclientset kubernetes.Interface,
	clientset csclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	csInformerFactory csinformers.SharedInformerFactory) *ClusterLabelController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(clusterLabelDelayBetweenRetries, clusterLabelDelayBetweenRetries)

	uc := &ClusterLabelController{
		kubeclientset: kubeclientset,
		csclientset:   clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, "ClusterLabel"),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, clusterLabelControllerName),
	}

	// Instantiate resource informers
	clusterLabelInformer := csInformerFactory.Containership().V3().ClusterLabels()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	log.Info(clusterLabelControllerName, ": Setting up event handlers")

	clusterLabelInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: uc.enqueueAllNodes,
		UpdateFunc: func(old, new interface{}) {
			oldClusterLabel := old.(*csv3.ClusterLabel)
			newClusterLabel := new.(*csv3.ClusterLabel)
			if oldClusterLabel.ResourceVersion == newClusterLabel.ResourceVersion {
				return
			}
			uc.enqueueAllNodes(new)
		},
		DeleteFunc: uc.enqueueAllNodes,
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: uc.enqueueNode,
		UpdateFunc: func(old, new interface{}) {
			oldNode := old.(*corev1.Node)
			newNode := new.(*corev1.Node)
			if oldNode.ResourceVersion == newNode.ResourceVersion {
				return
			}
			uc.enqueueNode(new)
		},
		DeleteFunc: uc.enqueueNode,
	})

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	uc.clusterLabelLister = clusterLabelInformer.Lister()
	uc.clusterLabelsSynced = clusterLabelInformer.Informer().HasSynced
	uc.nodeLister = nodeInformer.Lister()
	uc.nodesSynced = nodeInformer.Informer().HasSynced

	return uc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (uc *ClusterLabelController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer uc.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(clusterLabelControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		uc.clusterLabelsSynced,
		uc.nodesSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the clusterLabel controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(clusterLabelControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(uc.runWorker, time.Second, stopCh)
	}

	log.Info(clusterLabelControllerName, ": Started workers")
	<-stopCh
	log.Info(clusterLabelControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (uc *ClusterLabelController) runWorker() {
	for uc.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (uc *ClusterLabelController) processNextWorkItem() bool {
	obj, shutdown := uc.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer uc.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			uc.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		err := uc.nodeSyncHandler(key)
		return uc.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

func (uc *ClusterLabelController) handleErr(err error, key interface{}) error {
	if err == nil {
		uc.workqueue.Forget(key)
		return nil
	}

	if uc.workqueue.NumRequeues(key) < maxClusterLabelControllerRetries {
		uc.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing %q: %s. Has been resynced %v times", key, err.Error(), uc.workqueue.NumRequeues(key))
	}

	uc.workqueue.Forget(key)
	log.Infof("Dropping %q out of the queue: %v", key, err)
	return err
}

func (uc *ClusterLabelController) enqueueAllNodes(_ interface{}) {
	nodes, err := uc.nodeLister.List(labels.Everything())
	if err != nil {
		log.Error(err)
		return
	}

	for _, node := range nodes {
		uc.enqueueNode(node)
	}
}

// enqueueNode enqueues a node
func (uc *ClusterLabelController) enqueueNode(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

func (uc *ClusterLabelController) nodeSyncHandler(key string) error {
	_, name, _ := cache.SplitMetaNamespaceKey(key)

	node, err := uc.nodeLister.Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// Node is no longer around so nothing to do
			return nil
		}

		return errors.Wrapf(err, "getting node %s for cluster label reconciliation", name)
	}

	clusterLabels, err := uc.clusterLabelLister.ClusterLabels(constants.ContainershipNamespace).List(labels.Everything())
	if err != nil {
		return errors.Wrapf(err, "listing cluster labels to add to node %s", node.Name)
	}

	nodeLabels := buildLabelMapWithExactClusterLabels(node.Labels, clusterLabels)

	if tools.StringMapsAreEqual(node.Labels, nodeLabels) {
		// The labels haven't changed, so nothing to do
		return nil
	}

	// Update the node
	nodeCopy := node.DeepCopy()
	nodeCopy.Labels = nodeLabels

	_, err = uc.kubeclientset.CoreV1().Nodes().Update(nodeCopy)
	if err != nil {
		return errors.Wrap(err, "updating node with new labels")
	}

	return nil
}

// Given an existing label map and a list of cluster labels, return a new
// label map with:
//   1. All original non-cluster labels untouched
//   2. Cluster labels exactly matching the ones passed in
// Note that an empty map may be returned, but never a nil map.
func buildLabelMapWithExactClusterLabels(existingLabels map[string]string, clusterLabels []*csv3.ClusterLabel) map[string]string {
	// Build a new label map with all of the non-cluster-labels
	labels := make(map[string]string)
	for k, v := range existingLabels {
		if isContainershipClusterLabelKey(k) {
			continue
		}

		labels[k] = v
	}

	// Add all cluster labels to the node labels
	for _, clusterLabel := range clusterLabels {
		labels[clusterLabel.Spec.Key] = clusterLabel.Spec.Value
	}

	return labels
}

func isContainershipClusterLabelKey(key string) bool {
	return strings.HasPrefix(key, constants.ContainershipClusterLabelPrefix)
}
