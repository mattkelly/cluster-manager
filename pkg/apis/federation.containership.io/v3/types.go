package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster represents a cluster attached to Containership.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSpec `json:"spec"`
}

// ClusterSpec is the spec for a cluster.
type ClusterSpec struct {
	ID                 string `json:"id"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	OrganizationID     string `json:"organization_id"`
	Name               string `json:"name"`
	OwnerID            string `json:"owner_id"`
	APIServerAddress   string `json:"api_server_address"`
	WorkerNodesAddress string `json:"worker_nodes_address"`
	Environment        string `json:"environment"`
	Version            string `json:"version"`
	ProviderName       string `json:"provider_name"`

	// These are not part of the cluster response from cloud. Instead, a
	// separate labels request is made to load these.
	Labels []csv3.ClusterLabelSpec
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of Clusters.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Cluster `json:"items"`
}
