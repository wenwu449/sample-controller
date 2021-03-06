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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "k8s.io/sample-controller/pkg/apis/crdappcontroller/v1alpha1"
)

// UnderlayLister helps list Underlays.
type UnderlayLister interface {
	// List lists all Underlays in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Underlay, err error)
	// Underlays returns an object that can list and get Underlays.
	Underlays(namespace string) UnderlayNamespaceLister
	UnderlayListerExpansion
}

// underlayLister implements the UnderlayLister interface.
type underlayLister struct {
	indexer cache.Indexer
}

// NewUnderlayLister returns a new UnderlayLister.
func NewUnderlayLister(indexer cache.Indexer) UnderlayLister {
	return &underlayLister{indexer: indexer}
}

// List lists all Underlays in the indexer.
func (s *underlayLister) List(selector labels.Selector) (ret []*v1alpha1.Underlay, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Underlay))
	})
	return ret, err
}

// Underlays returns an object that can list and get Underlays.
func (s *underlayLister) Underlays(namespace string) UnderlayNamespaceLister {
	return underlayNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// UnderlayNamespaceLister helps list and get Underlays.
type UnderlayNamespaceLister interface {
	// List lists all Underlays in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Underlay, err error)
	// Get retrieves the Underlay from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Underlay, error)
	UnderlayNamespaceListerExpansion
}

// underlayNamespaceLister implements the UnderlayNamespaceLister
// interface.
type underlayNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Underlays in the indexer for a given namespace.
func (s underlayNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Underlay, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Underlay))
	})
	return ret, err
}

// Get retrieves the Underlay from the indexer for a given namespace and name.
func (s underlayNamespaceLister) Get(name string) (*v1alpha1.Underlay, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("underlay"), name)
	}
	return obj.(*v1alpha1.Underlay), nil
}
