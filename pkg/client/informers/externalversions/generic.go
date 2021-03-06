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

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
	containershipiov3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	provisioncontainershipiov3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=auth.containership.io, Version=v3
	case v3.SchemeGroupVersion.WithResource("authorizationroles"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ContainershipAuth().V3().AuthorizationRoles().Informer()}, nil
	case v3.SchemeGroupVersion.WithResource("authorizationrolebindings"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ContainershipAuth().V3().AuthorizationRoleBindings().Informer()}, nil

		// Group=containership.io, Version=v3
	case containershipiov3.SchemeGroupVersion.WithResource("clusterlabels"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Containership().V3().ClusterLabels().Informer()}, nil
	case containershipiov3.SchemeGroupVersion.WithResource("plugins"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Containership().V3().Plugins().Informer()}, nil
	case containershipiov3.SchemeGroupVersion.WithResource("registries"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Containership().V3().Registries().Informer()}, nil
	case containershipiov3.SchemeGroupVersion.WithResource("users"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Containership().V3().Users().Informer()}, nil

		// Group=provision.containership.io, Version=v3
	case provisioncontainershipiov3.SchemeGroupVersion.WithResource("clusterupgrades"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ContainershipProvision().V3().ClusterUpgrades().Informer()}, nil
	case provisioncontainershipiov3.SchemeGroupVersion.WithResource("nodepoollabels"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.ContainershipProvision().V3().NodePoolLabels().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
