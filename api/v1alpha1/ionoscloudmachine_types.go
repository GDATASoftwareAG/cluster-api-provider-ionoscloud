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

// IONOSCloudMachineSpec defines the desired state of IONOSCloudMachine
type IONOSCloudMachineSpec struct {
	DatacenterID string          `json:"datacenterID"`
	Cores        int             `json:"cores"`
	Ram          int             `json:"ram"`
	BootVolume   IONOSVolumeSpec `json:"bootVolume"`
	Lan          string          `json:"lan"`
	ProviderID   string          `json:"providerID"`
}

type IONOSVolumeSpec struct {
	Type  string `json:"type"`
	Size  int    `json:"size"`
	Image string `json:"image"`
}

// IONOSCloudMachineStatus defines the observed state of IONOSCloudMachine
type IONOSCloudMachineStatus struct {
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IONOSCloudMachine is the Schema for the ionoscloudmachines API
type IONOSCloudMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IONOSCloudMachineSpec   `json:"spec,omitempty"`
	Status IONOSCloudMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IONOSCloudMachineList contains a list of IONOSCloudMachine
type IONOSCloudMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IONOSCloudMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IONOSCloudMachine{}, &IONOSCloudMachineList{})
}
