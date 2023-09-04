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

// IONOSCloudClusterTemplateSpec defines the desired state of IONOSCloudClusterTemplate
type IONOSCloudClusterTemplateSpec struct {
	Template IONOSCloudClusterTemplateResource `json:"template"`
}

type IONOSCloudClusterTemplateResource struct {
	Spec IONOSCloudClusterSpec `json:"spec"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// IONOSCloudClusterTemplate is the Schema for the ionoscloudclustertemplates API
type IONOSCloudClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IONOSCloudClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// IONOSCloudClusterTemplateList contains a list of IONOSCloudClusterTemplate
type IONOSCloudClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IONOSCloudClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IONOSCloudClusterTemplate{}, &IONOSCloudClusterTemplateList{})
}
