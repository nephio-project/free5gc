/*
Copyright 2023.

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

// ImageConfigSpec defines the desired state of ImageConfig
type ImageConfigSpec struct {

	// ImagePaths is used to find the free5gc images
	ImagePaths map[string]string `json:"imagePaths,omitempty"`
}

// ImageConfigStatus defines the observed state of ImageConfig
type ImageConfigStatus struct {
	// Represnts the observation of a ImageConfig's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImageConfig is the Schema for the imageconfigs API
type ImageConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageConfigSpec   `json:"spec,omitempty"`
	Status ImageConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageConfigList contains a list of ImageConfig
type ImageConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageConfig{}, &ImageConfigList{})
}
