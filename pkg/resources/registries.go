package resources

import (
	"encoding/json"
	"fmt"

	"github.com/containership/cluster-manager/pkg/request"
	"github.com/containership/cluster-manager/pkg/resources/registry"
	"github.com/containership/cluster-manager/pkg/tools"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
)

// CsRegistries defines the Containership Cloud CsRegistries resource
type CsRegistries struct {
	cloudResource
	cache []csv3.RegistrySpec
}

// NewCsRegistries constructs a new CsRegistries
func NewCsRegistries() *CsRegistries {
	return &CsRegistries{
		cloudResource: cloudResource{
			endpoint: "/organizations/{{.OrganizationID}}/registries",
			service:  request.CloudServiceAPI,
		},
		cache: make([]csv3.RegistrySpec, 0),
	}
}

// UnmarshalToCache take the json returned from containership api
// and writes it to CsRegistries cache
func (rs *CsRegistries) UnmarshalToCache(bytes []byte) error {
	return json.Unmarshal(bytes, &rs.cache)
}

// Cache returns CsRegistries cache
func (rs *CsRegistries) Cache() []csv3.RegistrySpec {
	return rs.cache
}

// GetAuthToken return the AuthToken Generated by the registry generator
func (rs *CsRegistries) GetAuthToken(spec csv3.RegistrySpec) (csv3.AuthTokenDef, error) {
	generator := registry.New(spec.Provider, spec.Serveraddress, spec.Credentials)
	return generator.CreateAuthToken()
}

// IsEqual take a Registry Spec and compares it to a Registry to see if they are
// equal, returns an error if the objects are of an incorrect type
func (rs *CsRegistries) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(csv3.RegistrySpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type RegistrySpec")
	}

	user, ok := parentSpecObj.(*csv3.Registry)
	if !ok {
		return false, fmt.Errorf("The object is not of type Registry")
	}

	equal := spec.Description == user.Spec.Description &&
		spec.Organization == user.Spec.Organization &&
		spec.Email == user.Spec.Email &&
		spec.Serveraddress == user.Spec.Serveraddress &&
		spec.Provider == user.Spec.Provider &&
		spec.Owner == user.Spec.Owner

	if !equal {
		return false, nil
	}

	equal = tools.StringMapsAreEqual(spec.Credentials, user.Spec.Credentials)

	return equal, nil
}
