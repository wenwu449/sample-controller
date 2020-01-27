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

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "k8s.io/sample-controller/pkg/apis/crdappcontroller/v1alpha1"
)

// FakeUnderlays implements UnderlayInterface
type FakeUnderlays struct {
	Fake *FakeCrdappV1alpha1
	ns   string
}

var underlaysResource = schema.GroupVersionResource{Group: "crdapp.k8s.io", Version: "v1alpha1", Resource: "underlays"}

var underlaysKind = schema.GroupVersionKind{Group: "crdapp.k8s.io", Version: "v1alpha1", Kind: "Underlay"}

// Get takes name of the underlay, and returns the corresponding underlay object, and an error if there is any.
func (c *FakeUnderlays) Get(name string, options v1.GetOptions) (result *v1alpha1.Underlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(underlaysResource, c.ns, name), &v1alpha1.Underlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Underlay), err
}

// List takes label and field selectors, and returns the list of Underlays that match those selectors.
func (c *FakeUnderlays) List(opts v1.ListOptions) (result *v1alpha1.UnderlayList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(underlaysResource, underlaysKind, c.ns, opts), &v1alpha1.UnderlayList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.UnderlayList{ListMeta: obj.(*v1alpha1.UnderlayList).ListMeta}
	for _, item := range obj.(*v1alpha1.UnderlayList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested underlays.
func (c *FakeUnderlays) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(underlaysResource, c.ns, opts))

}

// Create takes the representation of a underlay and creates it.  Returns the server's representation of the underlay, and an error, if there is any.
func (c *FakeUnderlays) Create(underlay *v1alpha1.Underlay) (result *v1alpha1.Underlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(underlaysResource, c.ns, underlay), &v1alpha1.Underlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Underlay), err
}

// Update takes the representation of a underlay and updates it. Returns the server's representation of the underlay, and an error, if there is any.
func (c *FakeUnderlays) Update(underlay *v1alpha1.Underlay) (result *v1alpha1.Underlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(underlaysResource, c.ns, underlay), &v1alpha1.Underlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Underlay), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeUnderlays) UpdateStatus(underlay *v1alpha1.Underlay) (*v1alpha1.Underlay, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(underlaysResource, "status", c.ns, underlay), &v1alpha1.Underlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Underlay), err
}

// Delete takes name of the underlay and deletes it. Returns an error if one occurs.
func (c *FakeUnderlays) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(underlaysResource, c.ns, name), &v1alpha1.Underlay{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUnderlays) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(underlaysResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.UnderlayList{})
	return err
}

// Patch applies the patch and returns the patched underlay.
func (c *FakeUnderlays) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Underlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(underlaysResource, c.ns, name, pt, data, subresources...), &v1alpha1.Underlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Underlay), err
}
