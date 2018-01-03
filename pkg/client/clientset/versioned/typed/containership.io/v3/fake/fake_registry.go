/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	v3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRegistries implements RegistryInterface
type FakeRegistries struct {
	Fake *FakeContainershipV3
	ns   string
}

var registriesResource = schema.GroupVersionResource{Group: "containership.io", Version: "v3", Resource: "registries"}

var registriesKind = schema.GroupVersionKind{Group: "containership.io", Version: "v3", Kind: "Registry"}

// Get takes name of the registry, and returns the corresponding registry object, and an error if there is any.
func (c *FakeRegistries) Get(name string, options v1.GetOptions) (result *v3.Registry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(registriesResource, c.ns, name), &v3.Registry{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Registry), err
}

// List takes label and field selectors, and returns the list of Registries that match those selectors.
func (c *FakeRegistries) List(opts v1.ListOptions) (result *v3.RegistryList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(registriesResource, registriesKind, c.ns, opts), &v3.RegistryList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.RegistryList{}
	for _, item := range obj.(*v3.RegistryList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested registries.
func (c *FakeRegistries) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(registriesResource, c.ns, opts))

}

// Create takes the representation of a registry and creates it.  Returns the server's representation of the registry, and an error, if there is any.
func (c *FakeRegistries) Create(registry *v3.Registry) (result *v3.Registry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(registriesResource, c.ns, registry), &v3.Registry{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Registry), err
}

// Update takes the representation of a registry and updates it. Returns the server's representation of the registry, and an error, if there is any.
func (c *FakeRegistries) Update(registry *v3.Registry) (result *v3.Registry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(registriesResource, c.ns, registry), &v3.Registry{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Registry), err
}

// Delete takes name of the registry and deletes it. Returns an error if one occurs.
func (c *FakeRegistries) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(registriesResource, c.ns, name), &v3.Registry{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRegistries) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(registriesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v3.RegistryList{})
	return err
}

// Patch applies the patch and returns the patched registry.
func (c *FakeRegistries) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.Registry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(registriesResource, c.ns, name, data, subresources...), &v3.Registry{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Registry), err
}
