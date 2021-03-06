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

package v1alpha1

import (
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	crdappcontrollerv1alpha1 "k8s.io/sample-controller/pkg/apis/crdappcontroller/v1alpha1"
	versioned "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned"
	internalinterfaces "k8s.io/sample-controller/pkg/generated_crdapp/informers/externalversions/internalinterfaces"
	v1alpha1 "k8s.io/sample-controller/pkg/generated_crdapp/listers/crdappcontroller/v1alpha1"
)

// CrdappInformer provides access to a shared informer and lister for
// Crdapps.
type CrdappInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.CrdappLister
}

type crdappInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewCrdappInformer constructs a new informer for Crdapp type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCrdappInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCrdappInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredCrdappInformer constructs a new informer for Crdapp type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCrdappInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CrdappV1alpha1().Crdapps(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CrdappV1alpha1().Crdapps(namespace).Watch(options)
			},
		},
		&crdappcontrollerv1alpha1.Crdapp{},
		resyncPeriod,
		indexers,
	)
}

func (f *crdappInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCrdappInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *crdappInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&crdappcontrollerv1alpha1.Crdapp{}, f.defaultInformer)
}

func (f *crdappInformer) Lister() v1alpha1.CrdappLister {
	return v1alpha1.NewCrdappLister(f.Informer().GetIndexer())
}
