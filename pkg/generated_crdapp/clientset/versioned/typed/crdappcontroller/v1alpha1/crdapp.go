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

package v1alpha1

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "k8s.io/sample-controller/pkg/apis/crdappcontroller/v1alpha1"
	scheme "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned/scheme"
)

// CrdappsGetter has a method to return a CrdappInterface.
// A group's client should implement this interface.
type CrdappsGetter interface {
	Crdapps(namespace string) CrdappInterface
}

// CrdappInterface has methods to work with Crdapp resources.
type CrdappInterface interface {
	Create(*v1alpha1.Crdapp) (*v1alpha1.Crdapp, error)
	Update(*v1alpha1.Crdapp) (*v1alpha1.Crdapp, error)
	UpdateStatus(*v1alpha1.Crdapp) (*v1alpha1.Crdapp, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Crdapp, error)
	List(opts v1.ListOptions) (*v1alpha1.CrdappList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Crdapp, err error)
	CrdappExpansion
}

// crdapps implements CrdappInterface
type crdapps struct {
	client rest.Interface
	ns     string
}

// newCrdapps returns a Crdapps
func newCrdapps(c *CrdappV1alpha1Client, namespace string) *crdapps {
	return &crdapps{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the crdapp, and returns the corresponding crdapp object, and an error if there is any.
func (c *crdapps) Get(name string, options v1.GetOptions) (result *v1alpha1.Crdapp, err error) {
	result = &v1alpha1.Crdapp{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("crdapps").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Crdapps that match those selectors.
func (c *crdapps) List(opts v1.ListOptions) (result *v1alpha1.CrdappList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CrdappList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("crdapps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested crdapps.
func (c *crdapps) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("crdapps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a crdapp and creates it.  Returns the server's representation of the crdapp, and an error, if there is any.
func (c *crdapps) Create(crdapp *v1alpha1.Crdapp) (result *v1alpha1.Crdapp, err error) {
	result = &v1alpha1.Crdapp{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("crdapps").
		Body(crdapp).
		Do().
		Into(result)
	return
}

// Update takes the representation of a crdapp and updates it. Returns the server's representation of the crdapp, and an error, if there is any.
func (c *crdapps) Update(crdapp *v1alpha1.Crdapp) (result *v1alpha1.Crdapp, err error) {
	result = &v1alpha1.Crdapp{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("crdapps").
		Name(crdapp.Name).
		Body(crdapp).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *crdapps) UpdateStatus(crdapp *v1alpha1.Crdapp) (result *v1alpha1.Crdapp, err error) {
	result = &v1alpha1.Crdapp{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("crdapps").
		Name(crdapp.Name).
		SubResource("status").
		Body(crdapp).
		Do().
		Into(result)
	return
}

// Delete takes name of the crdapp and deletes it. Returns an error if one occurs.
func (c *crdapps) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("crdapps").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *crdapps) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("crdapps").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched crdapp.
func (c *crdapps) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Crdapp, err error) {
	result = &v1alpha1.Crdapp{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("crdapps").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}