package resources

import (
	"encoding/json"

	"github.com/pkg/errors"

	cscloud "github.com/containership/csctl/cloud"
	apitypes "github.com/containership/csctl/cloud/api/types"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	csfedv3 "github.com/containership/cluster-manager/pkg/apis/federation.containership.io/v3"
)

// CsClusters defines the Containership Cloud Clusters resource
type CsClusters struct {
	cloudResource
	cache []csfedv3.ClusterSpec
}

// NewCsClusters constructs a new CsClusters
func NewCsClusters(cloud cscloud.Interface) *CsClusters {
	cache := make([]csfedv3.ClusterSpec, 0)
	return &CsClusters{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (cp *CsClusters) Sync() error {
	clusters, err := cp.cloud.API().Clusters(cp.organizationID).List()
	if err != nil {
		return errors.Wrap(err, "listing clusters")
	}

	// Use an intermediate type that will marshal to the same json as a
	// csfedv3.Cluster. This allows us to easily transform from the
	// conglomerate of cloud responses to the CR type by round-tripping through
	// the json serializer.
	type clusterWithLabels struct {
		Cluster apitypes.Cluster        `json:",inline"`
		Labels  []apitypes.ClusterLabel `json:"labels"`
	}

	clustersWithLabels := make([]clusterWithLabels, len(clusters))
	for i, cluster := range clusters {
		labels, err := cp.cloud.API().ClusterLabels(cp.organizationID, string(cluster.ID)).List()
		if err != nil {
			return errors.Wrapf(err, "getting labels for cluster %q", cluster.ID)
		}

		clustersWithLabels[i] = clusterWithLabels{
			Cluster: cluster,
			Labels:  labels,
		}
	}

	data, err := json.Marshal(clustersWithLabels)
	if err != nil {
		return err
	}

	json.Unmarshal(data, &cp.cache)

	return nil
}

// Cache return the containership clusters cache
func (cp *CsClusters) Cache() []csfedv3.ClusterSpec {
	return cp.cache
}

// IsEqual compares a ClusterSpec to another Cluster
func (cp *CsClusters) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(csfedv3.ClusterSpec)
	if !ok {
		return false, errors.New("object is not of type ClusterSpec")
	}

	cluster, ok := parentSpecObj.(*csfedv3.Cluster)
	if !ok {
		return false, errors.New("object is not of type Cluster")
	}

	equal := cluster.Spec.ID == spec.ID &&
		cluster.Spec.CreatedAt == spec.CreatedAt &&
		cluster.Spec.UpdatedAt == spec.UpdatedAt &&
		cluster.Spec.OrganizationID == spec.OrganizationID &&
		cluster.Spec.Name == spec.Name &&
		cluster.Spec.OwnerID == spec.OwnerID &&
		cluster.Spec.APIServerAddress == spec.APIServerAddress &&
		cluster.Spec.WorkerNodesAddress == spec.WorkerNodesAddress &&
		cluster.Spec.Environment == spec.Environment &&
		cluster.Spec.Version == spec.Version &&
		cluster.Spec.ProviderName == spec.ProviderName

	if !equal {
		return false, nil
	}

	return clusterLabelSlicesAreEqual(cluster.Spec.Labels, spec.Labels), nil
}

func clusterLabelSlicesAreEqual(sliceA, sliceB []csv3.ClusterLabelSpec) bool {
	if len(sliceA) != len(sliceB) ||
		sliceA == nil && sliceB != nil ||
		sliceB == nil && sliceA != nil {
		return false
	}

	// TODO this logic is borrowed from CsClusterLabels.IsEqual(). Not sure why
	// the IsEqual() functions have receivers...
	for i, l := range sliceA {
		equal := l.ID == sliceB[i].ID &&
			l.CreatedAt == sliceB[i].CreatedAt &&
			l.UpdatedAt == sliceB[i].UpdatedAt &&
			l.Key == sliceB[i].Key &&
			l.Value == sliceB[i].Value

		if !equal {
			return false
		}
	}

	return true
}
