package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pcsv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
	"github.com/containership/cluster-manager/pkg/constants"
)

var emptyNodePoolLabelSpec = pcsv3.NodePoolLabelSpec{}
var emptyNodePoolLabel = &pcsv3.NodePoolLabel{}

var nodePoolLabel1 = &pcsv3.NodePoolLabel{
	ObjectMeta: metav1.ObjectMeta{
		Name: "name0",
	},
	Spec: pcsv3.NodePoolLabelSpec{
		ID:         "1",
		CreatedAt:  "123",
		UpdatedAt:  "456",
		Key:        constants.ContainershipNodePoolLabelPrefix + "key1",
		Value:      "value1",
		NodePoolID: "4321",
	},
}

func TestNodePoolLabelIsEqual(t *testing.T) {
	c := NewCsNodePoolLabels(nil)

	_, err := c.IsEqual("wrong type", emptyNodePoolLabel)
	assert.Error(t, err, "bad spec type")

	_, err = c.IsEqual(emptyNodePoolLabelSpec, "wrong type")
	assert.Error(t, err, "bad parent type")

	eq, err := c.IsEqual(emptyNodePoolLabelSpec, emptyNodePoolLabel)
	assert.NoError(t, err)
	assert.True(t, eq, "both empty")

	eq, err = c.IsEqual(emptyNodePoolLabelSpec, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "spec only empty")

	eq, err = c.IsEqual(nodePoolLabel1.Spec, emptyNodePoolLabel)
	assert.NoError(t, err)
	assert.False(t, eq, "parent only empty")

	same := nodePoolLabel1.DeepCopy().Spec
	eq, err = c.IsEqual(same, nodePoolLabel1)
	assert.NoError(t, err)
	assert.True(t, eq, "copied spec")

	diff := nodePoolLabel1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "different ID")

	diff = nodePoolLabel1.DeepCopy().Spec
	diff.CreatedAt = "different"
	eq, err = c.IsEqual(diff, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "different created_at")

	diff = nodePoolLabel1.DeepCopy().Spec
	diff.UpdatedAt = "different"
	eq, err = c.IsEqual(diff, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "different updated_at")

	diff = nodePoolLabel1.DeepCopy().Spec
	diff.Key = "different"
	eq, err = c.IsEqual(diff, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "different key")

	diff = nodePoolLabel1.DeepCopy().Spec
	diff.Value = "different"
	eq, err = c.IsEqual(diff, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "different value")

	diff = nodePoolLabel1.DeepCopy().Spec
	diff.NodePoolID = "different"
	eq, err = c.IsEqual(diff, nodePoolLabel1)
	assert.NoError(t, err)
	assert.False(t, eq, "different node Pool ID")
}

func TestNodePoolLabelsCache(t *testing.T) {
	nodePoolLabelBytes := []byte(`[{
	"id" : "1234",
	"created_at" : "123",
	"updated_at" : "456",
	"key" : "key1",
	"value" : "value1",
	"node_pool_id" : "4321"
}]`)

	u := NewCsNodePoolLabels(nil)

	err := json.Unmarshal(nodePoolLabelBytes, &u.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := u.Cache()
	assert.Equal(t, u.cache, c)
}
