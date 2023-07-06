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

// IONOSCloudMachineTemplateSpec defines the desired state of IONOSCloudMachineTemplate
type IONOSCloudMachineTemplateSpec struct {
	Template IONOSCloudMachineTemplateResource `json:"template"`
}

type IONOSCloudMachineTemplateResource struct {
	Template IONOSCloudMachineSpec `json:"template"`
}

//+kubebuilder:object:root=true

// IONOSCloudMachineTemplate is the Schema for the ionoscloudmachinetemplates API
type IONOSCloudMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IONOSCloudMachineTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// IONOSCloudMachineTemplateList contains a list of IONOSCloudMachineTemplate
type IONOSCloudMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IONOSCloudMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IONOSCloudMachineTemplate{}, &IONOSCloudMachineTemplateList{})
}
