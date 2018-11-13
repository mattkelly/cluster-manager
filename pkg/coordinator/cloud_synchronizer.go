package coordinator

import (
	"time"

	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	"github.com/containership/cluster-manager/pkg/k8sutil"
	"github.com/containership/cluster-manager/pkg/log"
	synccontroller "github.com/containership/cluster-manager/pkg/resources/sync_controller"
)

// CloudSynchronizer synchronizes Containership Cloud resources
// into our Kubernetes CRDs.
type CloudSynchronizer struct {
	syncControllers []synccontroller.Interface
	syncStopCh      chan struct{}
	stopped         bool
}

// NewCloudSynchronizer constructs a new CloudSynchronizer.
func NewCloudSynchronizer(csInformerFactory csinformers.SharedInformerFactory) *CloudSynchronizer {
	return &CloudSynchronizer{
		syncStopCh: make(chan struct{}),
	}
}

// Run kicks off cloud sync routines.
func (s *CloudSynchronizer) Run() {
	log.Info("Running CloudSynchronizer")
	for _, sc := range s.syncControllers {
		go sc.SyncWithCloud(s.syncStopCh)
	}
}

// RegisterController registers a SyncController
func (s *CloudSynchronizer) RegisterController(sc synccontroller.Interface) {
	s.syncControllers = append(s.syncControllers, sc)
}

// RequestTerminate requests that all Containership resources be deleted from
// the cluster. It kicks off a goroutine to marks resources for deletion and
// returns without blocking.
func (s *CloudSynchronizer) RequestTerminate() {
	// Stop synchronizing cloud resources
	s.stopAllSyncRoutines()

	go cleanupAllContainershipManagedResources()
}

// stopAllSyncRoutines stops all cloud synchronization but does not clean up
// any resources.
func (s *CloudSynchronizer) stopAllSyncRoutines() {
	if s.stopped {
		log.Info("CloudSynchronizer already stopped")
		return
	}

	log.Info("Stopping CloudSynchronizer")
	close(s.syncStopCh)
	s.stopped = true
}

// cleanupAllContainershipManagedResources performs a best-effort attempt to
// clean up all CS resources by deleting all CS CRDs, cleaning up Containership
// service accouns, and then deleting the core Containership namespace after a
// delay, which should result in the agent and coordinator being killed.
func cleanupAllContainershipManagedResources() {
	tryDeleteAllContainershipCRDs()
	tryDeleteAllContainershipServiceAccounts()

	// TODO it would be great if we could avoid this arbitrary heuristic here.
	// Sleep for a little to give k8s enough time to attempt to delete all
	// resources. This should provide enough time for the agents to see the
	// CRD deletions (which should happen first) and clean up any on-host changes
	// such as authorized_keys.
	time.Sleep(time.Minute)

	tryDeleteAllContainershipNamespaces()
}

// tryDeleteAllContainershipCRDs tries to delete all CRDs managed by us. This will cause cascading delete to clean up all Containership-managed resources.
func tryDeleteAllContainershipCRDs() {
	crdList, err := k8sutil.ExtensionsAPI().GetContainershipCRDs()
	if err != nil {
		log.Error("Could not list CRDs for cleanup:", err.Error())
		return
	}

	for _, crd := range crdList.Items {
		log.Info("Deleting CRD", crd.Name)
		err := k8sutil.ExtensionsAPI().DeleteCRD(crd.Name)
		if err != nil {
			log.Errorf("Could not delete CRD %s: %s", crd.Name, err.Error())
		}
	}
}

// tryDeleteAllContainershipServiceAccounts tries to delete all
// service accounts in each namespace that are containership managed
func tryDeleteAllContainershipServiceAccounts() {
	nsList, err := k8sutil.API().GetNamespaces()
	if err != nil {
		log.Error("Could not list namespaces for cleanup:", err.Error())
		return
	}

	for _, ns := range nsList.Items {
		log.Info("Deleting managed Service Accounts in ", ns.Name)
		err := k8sutil.API().DeleteContainershipServiceAccounts(ns.Name)
		if err != nil {
			log.Errorf("Could not delete service accounts in namespace %s: %s", ns.Name, err.Error())
		}
	}
}

// tryDeleteAllContainershipNamespaces tries to delete all
// namespaces managed by us. This will cause all resources
// belonging to the namespace, including the pod that this is
// running in, to be deleted.
func tryDeleteAllContainershipNamespaces() {
	nsList, err := k8sutil.API().GetContainershipNamespaces()
	if err != nil {
		log.Error("Could not list namespaces for cleanup:", err.Error())
		return
	}

	for _, ns := range nsList.Items {
		log.Info("Deleting Namespace", ns.Name)
		err := k8sutil.API().DeleteNamespace(ns.Name)
		if err != nil {
			log.Errorf("Could not delete Namespace %s: %s", ns.Name, err.Error())
		}
	}
}
