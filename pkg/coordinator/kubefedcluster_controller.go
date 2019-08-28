package coordinator

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubefedv1beta1 "github.com/containership/cluster-manager/pkg/apis/core.kubefed.io/v1beta1"

	kubefedlisters "github.com/containership/cluster-manager/pkg/client/listers/core.kubefed.io/v1beta1"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	csfedv3 "github.com/containership/cluster-manager/pkg/apis/federation.containership.io/v3"

	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	csfedlisters "github.com/containership/cluster-manager/pkg/client/listers/federation.containership.io/v3"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/tools"
)

const (
	kubeFedClusterControllerName = "KubeFedClusterController"

	kubeFedClusterDelayBetweenRetries = 30 * time.Second

	maxKubeFedClusterControllerRetries = 10
)

// KubeFedClusterController is the controller implementation for the containership
// upgrading clusters to a users desired kubernetes version
type KubeFedClusterController struct {
	kubeclientset kubernetes.Interface
	csclientset   csclientset.Interface

	clusterLister  csfedlisters.ClusterLister
	clustersSynced cache.InformerSynced

	kubeFedClusterLister  kubefedlisters.KubeFedClusterLister
	kubeFedClustersSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewKubeFedClusterController returns a new clusterLabel controller
func NewKubeFedClusterController(kubeclientset kubernetes.Interface,
	clientset csclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	csInformerFactory csinformers.SharedInformerFactory) *KubeFedClusterController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(kubeFedClusterDelayBetweenRetries, kubeFedClusterDelayBetweenRetries)

	c := &KubeFedClusterController{
		kubeclientset: kubeclientset,
		csclientset:   clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, "Cluster"),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, kubeFedClusterControllerName),
	}

	// Instantiate resource informers
	clusterInformer := csInformerFactory.ContainershipFederation().V3().Clusters()
	kubeFedClusterInformer := csInformerFactory.KubeFedCore().V1beta1().KubeFedClusters()

	log.Info(kubeFedClusterControllerName, ": Setting up event handlers")

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueCluster,
		UpdateFunc: func(old, new interface{}) {
			oldCluster := old.(*csfedv3.Cluster)
			newCluster := new.(*csfedv3.Cluster)
			if oldCluster.ResourceVersion == newCluster.ResourceVersion {
				return
			}
			c.enqueueCluster(new)
		},
		// we don't need to listen for deletes because the finalizer logic
		// depends only on update events.
	})

	kubeFedClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueClusterForKubeFedCluster,
		UpdateFunc: func(old, new interface{}) {
			oldCluster := old.(*kubefedv1beta1.KubeFedCluster)
			newCluster := new.(*kubefedv1beta1.KubeFedCluster)
			if oldCluster.ResourceVersion == newCluster.ResourceVersion {
				return
			}
			c.enqueueClusterForKubeFedCluster(new)
		},
		DeleteFunc: c.enqueueClusterForKubeFedCluster,
	})

	c.clusterLister = clusterInformer.Lister()
	c.clustersSynced = clusterInformer.Informer().HasSynced
	c.kubeFedClusterLister = kubeFedClusterInformer.Lister()
	c.kubeFedClustersSynced = kubeFedClusterInformer.Informer().HasSynced

	return c
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *KubeFedClusterController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(kubeFedClusterControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		c.clustersSynced,
		c.kubeFedClustersSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the clusterLabel controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(kubeFedClusterControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info(kubeFedClusterControllerName, ": Started workers")
	<-stopCh
	log.Info(kubeFedClusterControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *KubeFedClusterController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (c *KubeFedClusterController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		err := c.clusterSyncHandler(key)
		return c.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

func (c *KubeFedClusterController) handleErr(err error, key interface{}) error {
	if err == nil {
		c.workqueue.Forget(key)
		return nil
	}

	if c.workqueue.NumRequeues(key) < maxKubeFedClusterControllerRetries {
		c.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing %q: %s. Has been resynced %v times", key, err.Error(), c.workqueue.NumRequeues(key))
	}

	c.workqueue.Forget(key)
	log.Infof("Dropping %q out of the queue: %v", key, err)
	return err
}

// enqueueCluster enqueues a cluster
func (c *KubeFedClusterController) enqueueCluster(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	c.workqueue.AddRateLimited(key)
}

// enqueueClusterForKubeFedCluster enqueues the Containership Cluster for a
// KubeFedCluster Since the name of the KubeFedCluster is equal to the name of
// the Containership cluster and the namespace for Clusters is always the same,
// we can just build the key manually here.
func (c *KubeFedClusterController) enqueueClusterForKubeFedCluster(obj interface{}) {
	cluster := obj.(*kubefedv1beta1.KubeFedCluster)
	key := fmt.Sprintf("%s/%s", constants.ContainershipNamespace, cluster.Name)
	c.workqueue.AddRateLimited(key)
}

func (c *KubeFedClusterController) clusterSyncHandler(key string) error {
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)

	if namespace != constants.ContainershipNamespace {
		// Should never happen
		log.Infof("Ignoring Containership Cluster %s in namespace %s", name, namespace)
		return nil
	}

	cluster, err := c.clusterLister.Clusters(constants.ContainershipNamespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// Cluster CR is no longer around so nothing to do
			return nil
		}

		return errors.Wrapf(err, "getting Cluster CR %s for KubeFedCluster reconciliation", name)
	}

	// If this cluster doesn't belong to a federation, then we can ignore it.
	federationName := getFederationNameForCluster(cluster.Spec)
	thisFederation := env.FederationName()
	if federationName != thisFederation {
		log.Debugf("%s: ignoring Cluster %s that does not belong to federation %s",
			kubeFedClusterControllerName, cluster.Name, thisFederation)
		return nil
	}

	if !cluster.DeletionTimestamp.IsZero() {
		// This Cluster is marked for deletion. We must delete the
		// dependent/generated KubeFedCluster we may have created and remove the
		// finalizer on the CR before Kubernetes will actually delete it.
		// The name of the generated KubeFedCluster will be equal to the name of this
		// Cluster.
		err := c.deleteKubeFedClusterIfExists(name)
		if err != nil {
			return errors.Wrapf(err, "cleaning up generated KubeFedCluster %s", name)
		}

		// Either we successfully removed the dependent KubeFedCluster or it
		// didn't exist, so now remove the finalizer and let Kubernetes take
		// over.
		clusterCopy := cluster.DeepCopy()
		clusterCopy.Finalizers = tools.RemoveStringFromSlice(clusterCopy.Finalizers, constants.ClusterFinalizerName)

		_, err = c.csclientset.ContainershipFederationV3().Clusters(constants.ContainershipNamespace).Update(clusterCopy)
		if err != nil {
			return errors.Wrap(err, "updating Cluster with finalizer removed")
		}

		return nil
	}

	// KubeFedClusters we create always live in the kubefed namespace and
	// always have the same name as the owning Cluster
	kfCluster, err := c.kubeFedClusterLister.KubeFedClusters(constants.KubeFedNamespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// We don't have a matching KubeFedCluster, so create it
			log.Infof("%s: Creating missing KubeFedCluster %s", kubeFedClusterControllerName, name)
			kfClusterNew := kubeFedClusterFromClusterSpec(cluster.Spec)
			_, err = c.csclientset.KubeFedCoreV1beta1().KubeFedClusters(constants.KubeFedNamespace).Create(&kfClusterNew)
			return errors.Wrapf(err, "creating new KubeFedCluster %q", kfClusterNew.Name)
		}
	}

	// The KubeFedCluster exists - make sure it matches the Cluster
	// TODO for now just always update
	// TODO can't use kubeFedClusterFromClusterSpec() to build an entirely new
	// cluster because (unlike other resources for which we do this) it
	// complains about the missing ResourceVersion on updates. Copy and modify
	// instead.
	kfClusterCopy := kfCluster.DeepCopy()

	additionalLabels := labelMapFromClusterLabels(cluster.Spec.Labels)
	kfClusterCopy.Labels = constants.BuildContainershipLabelMap(additionalLabels)
	kfClusterCopy.Spec = kubefedv1beta1.KubeFedClusterSpec{
		APIEndpoint: getContainershipProxyAddressForCluster(cluster.Spec.ID),
		// This secret must live in the kubefed core namespace
		SecretRef: kubefedv1beta1.LocalSecretReference{
			// TODO env var
			Name: "containership-token",
		},
	}

	_, err = c.csclientset.KubeFedCoreV1beta1().KubeFedClusters(constants.KubeFedNamespace).Update(kfClusterCopy)
	return err
}

func (c *KubeFedClusterController) deleteKubeFedClusterIfExists(name string) error {
	_, err := c.kubeFedClusterLister.KubeFedClusters(constants.KubeFedNamespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// It does exist, so delete it.
	return c.csclientset.KubeFedCoreV1beta1().KubeFedClusters(constants.KubeFedNamespace).Delete(name, &metav1.DeleteOptions{})
}

func kubeFedClusterFromClusterSpec(cluster csfedv3.ClusterSpec) kubefedv1beta1.KubeFedCluster {
	additionalLabels := labelMapFromClusterLabels(cluster.Labels)

	return kubefedv1beta1.KubeFedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cluster.ID,
			Labels: additionalLabels,
		},
		Spec: kubefedv1beta1.KubeFedClusterSpec{
			APIEndpoint: getContainershipProxyAddressForCluster(cluster.ID),
			// This secret must live in the kubefed core namespace
			SecretRef: kubefedv1beta1.LocalSecretReference{
				// TODO env var
				Name: "containership-token",
			},
		},
	}
}

func labelMapFromClusterLabels(clusterLabels []csv3.ClusterLabelSpec) map[string]string {
	if clusterLabels == nil {
		return nil
	}

	m := map[string]string{}
	for _, label := range clusterLabels {
		m[label.Key] = label.Value
	}

	return m
}

func getFederationNameForCluster(cluster csfedv3.ClusterSpec) string {
	for _, label := range cluster.Labels {
		if label.Key == constants.ContainershipFederationNameLabelKey {
			return label.Value
		}
	}

	return ""
}

func getContainershipProxyAddressForCluster(clusterID string) string {
	return fmt.Sprintf("%s/v3/organizations/%s/clusters/%s/k8sapi/proxy",
		env.ProxyBaseURL(), env.OrganizationID(), clusterID)
}
