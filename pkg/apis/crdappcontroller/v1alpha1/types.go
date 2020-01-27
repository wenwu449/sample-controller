/*
Copyright 2017 The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Crdapp is a specification for a Crdapp resource
type Crdapp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CrdappSpec   `json:"spec"`
	Status CrdappStatus `json:"status"`
}

// CrdappSpec is the spec for a Crdapp resource
type CrdappSpec struct {
	Type string `json:"type"`
	Version       string `json:"version"`
}

// CrdappStatus is the status for a Crdapp resource
type CrdappStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CrdappList is a list of Crdapp resources
type CrdappList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Crdapp `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Underlay is a specification for a Underlay resource
type Underlay struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UnderlaySpec   `json:"spec"`
	Status UnderlayStatus `json:"status"`
}

// UnderlaySpec is the spec for a Underlay resource
type UnderlaySpec struct {
	Type string   `json:"type"`
	Version       string `json:"version"`
	KubeConfig string `json:"kubeConfig"`
}

// UnderlayStatus is the status for a Underlay resource
type UnderlayStatus struct {
	AvailableApps int32 `json:"availableApps"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UnderlayList is a list of Underlay resources
type UnderlayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Underlay `json:"items"`
}
