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

// Code generated by informer-gen. DO NOT EDIT.

package v3

import (
	time "time"

	containershipiov3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	versioned "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	internalinterfaces "github.com/containership/cloud-agent/pkg/client/informers/externalversions/internalinterfaces"
	v3 "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// RegistryInformer provides access to a shared informer and lister for
// Registries.
type RegistryInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.RegistryLister
}

type registryInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewRegistryInformer constructs a new informer for Registry type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRegistryInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRegistryInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredRegistryInformer constructs a new informer for Registry type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRegistryInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ContainershipV3().Registries(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ContainershipV3().Registries(namespace).Watch(options)
			},
		},
		&containershipiov3.Registry{},
		resyncPeriod,
		indexers,
	)
}

func (f *registryInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRegistryInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *registryInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&containershipiov3.Registry{}, f.defaultInformer)
}

func (f *registryInformer) Lister() v3.RegistryLister {
	return v3.NewRegistryLister(f.Informer().GetIndexer())
}
