/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package versioned

import (
	containershipauthv3 "github.com/containership/cluster-manager/pkg/client/clientset/versioned/typed/auth.containership.io/v3"
	containershipv3 "github.com/containership/cluster-manager/pkg/client/clientset/versioned/typed/containership.io/v3"
	kubefedcorev1beta1 "github.com/containership/cluster-manager/pkg/client/clientset/versioned/typed/core.kubefed.io/v1beta1"
	containershipfederationv3 "github.com/containership/cluster-manager/pkg/client/clientset/versioned/typed/federation.containership.io/v3"
	containershipprovisionv3 "github.com/containership/cluster-manager/pkg/client/clientset/versioned/typed/provision.containership.io/v3"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	ContainershipAuthV3() containershipauthv3.ContainershipAuthV3Interface
	ContainershipV3() containershipv3.ContainershipV3Interface
	KubeFedCoreV1beta1() kubefedcorev1beta1.KubeFedCoreV1beta1Interface
	ContainershipFederationV3() containershipfederationv3.ContainershipFederationV3Interface
	ContainershipProvisionV3() containershipprovisionv3.ContainershipProvisionV3Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	containershipAuthV3       *containershipauthv3.ContainershipAuthV3Client
	containershipV3           *containershipv3.ContainershipV3Client
	kubeFedCoreV1beta1        *kubefedcorev1beta1.KubeFedCoreV1beta1Client
	containershipFederationV3 *containershipfederationv3.ContainershipFederationV3Client
	containershipProvisionV3  *containershipprovisionv3.ContainershipProvisionV3Client
}

// ContainershipAuthV3 retrieves the ContainershipAuthV3Client
func (c *Clientset) ContainershipAuthV3() containershipauthv3.ContainershipAuthV3Interface {
	return c.containershipAuthV3
}

// ContainershipV3 retrieves the ContainershipV3Client
func (c *Clientset) ContainershipV3() containershipv3.ContainershipV3Interface {
	return c.containershipV3
}

// KubeFedCoreV1beta1 retrieves the KubeFedCoreV1beta1Client
func (c *Clientset) KubeFedCoreV1beta1() kubefedcorev1beta1.KubeFedCoreV1beta1Interface {
	return c.kubeFedCoreV1beta1
}

// ContainershipFederationV3 retrieves the ContainershipFederationV3Client
func (c *Clientset) ContainershipFederationV3() containershipfederationv3.ContainershipFederationV3Interface {
	return c.containershipFederationV3
}

// ContainershipProvisionV3 retrieves the ContainershipProvisionV3Client
func (c *Clientset) ContainershipProvisionV3() containershipprovisionv3.ContainershipProvisionV3Interface {
	return c.containershipProvisionV3
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.containershipAuthV3, err = containershipauthv3.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.containershipV3, err = containershipv3.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.kubeFedCoreV1beta1, err = kubefedcorev1beta1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.containershipFederationV3, err = containershipfederationv3.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.containershipProvisionV3, err = containershipprovisionv3.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.containershipAuthV3 = containershipauthv3.NewForConfigOrDie(c)
	cs.containershipV3 = containershipv3.NewForConfigOrDie(c)
	cs.kubeFedCoreV1beta1 = kubefedcorev1beta1.NewForConfigOrDie(c)
	cs.containershipFederationV3 = containershipfederationv3.NewForConfigOrDie(c)
	cs.containershipProvisionV3 = containershipprovisionv3.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.containershipAuthV3 = containershipauthv3.New(c)
	cs.containershipV3 = containershipv3.New(c)
	cs.kubeFedCoreV1beta1 = kubefedcorev1beta1.New(c)
	cs.containershipFederationV3 = containershipfederationv3.New(c)
	cs.containershipProvisionV3 = containershipprovisionv3.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
