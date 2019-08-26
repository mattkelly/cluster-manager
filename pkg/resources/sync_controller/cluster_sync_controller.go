package synccontroller

import (
	"fmt"

	cscloud "github.com/containership/csctl/cloud"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	csfedv3 "github.com/containership/cluster-manager/pkg/apis/federation.containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	csfedlisters "github.com/containership/cluster-manager/pkg/client/listers/federation.containership.io/v3"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
)

// ClusterSyncController is the implementation for syncing Cluster CRDs
type ClusterSyncController struct {
	*syncController

	lister        csfedlisters.ClusterLister
	cloudResource *resources.CsClusters
}

const (
	clusterSyncControllerName = "ClusterSyncController"
)

// NewCluster returns a ClusterSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewCluster(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *ClusterSyncController {
	clusterInformer := csInformerFactory.ContainershipFederation().V3().Clusters()

	clusterInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &ClusterSyncController{
		syncController: &syncController{
			name:      clusterSyncControllerName,
			clientset: clientset,
			synced:    clusterInformer.Informer().HasSynced,
			informer:  clusterInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, clusterSyncControllerName),
		},

		lister:        clusterInformer.Lister(),
		cloudResource: resources.NewCsClusters(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *ClusterSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *ClusterSyncController) doSync() {
	log.Debug("Sync Clusters")
	// makes a request to containership api and write results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("Clusters failed to sync: ", err.Error())
		return
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.ID] = cloudItem

		// Try to find cloud item in CR cache
		item, err := c.informer.GetIndexer().ByIndex(tools.IndexByIDFunctionName, cloudItem.ID)
		if err == nil && len(item) == 0 {
			log.Debugf("Cloud Cluster %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("Cluster Create failed: ", err.Error())
			}
			continue
		}

		clusterCR := item[0]
		if equal, err := c.cloudResource.IsEqual(cloudItem, clusterCR); err == nil && !equal {
			log.Debugf("Cloud Cluster %s does not match CR - updating", cloudItem.ID)
			err = c.Update(cloudItem, clusterCR)
			if err != nil {
				log.Error("Cluster Update failed: ", err.Error())
			}
			continue
		}
	}

	allCRs, err := c.lister.List(labels.NewSelector())
	if err != nil {
		log.Error(err)
		return
	}

	// Find CRs that do not exist in cloud
	for _, u := range allCRs {
		if _, exists := cloudCacheByID[u.Name]; !exists {
			log.Debugf("CR Cluster %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("Cluster Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a cluster spec in cache and creates the CRD
func (c *ClusterSyncController) Create(clusterSpec csfedv3.ClusterSpec) error {
	cluster, err := c.clientset.ContainershipFederationV3().Clusters(constants.ContainershipNamespace).Create(&csfedv3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterSpec.ID,
		},
		Spec: clusterSpec,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(cluster, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes a cluster spec and updates the associated Cluster CR spec
// with the new values
func (c *ClusterSyncController) Update(clusterSpec csfedv3.ClusterSpec, obj interface{}) error {
	cluster, ok := obj.(*csfedv3.Cluster)
	if !ok {
		return fmt.Errorf("Error trying to use a non Cluster CR object to update a Cluster CR")
	}

	c.recorder.Event(cluster, corev1.EventTypeNormal, "ClusterUpdate",
		"Detected change in Cloud, updating")

	cCopy := cluster.DeepCopy()
	cCopy.Spec = clusterSpec

	_, err := c.clientset.ContainershipFederationV3().Clusters(constants.ContainershipNamespace).Update(cCopy)

	if err != nil {
		c.recorder.Eventf(cluster, corev1.EventTypeWarning, "ClusterUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes a name or the CRD and deletes it
func (c *ClusterSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipFederationV3().Clusters(namespace).Delete(name, &metav1.DeleteOptions{})
}
