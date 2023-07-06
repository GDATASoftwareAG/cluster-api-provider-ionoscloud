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
	"sigs.k8s.io/cluster-api/api/v1beta1"
)

// IONOSCloudClusterSpec defines the desired state of IONOSCloudCluster
type IONOSCloudClusterSpec struct {
	ControlPlaneEndpoint v1beta1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
}

// IONOSCloudClusterStatus defines the observed state of IONOSCloudCluster
type IONOSCloudClusterStatus struct {
	Ready string `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IONOSCloudCluster is the Schema for the ionoscloudclusters API
type IONOSCloudCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IONOSCloudClusterSpec   `json:"spec,omitempty"`
	Status IONOSCloudClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IONOSCloudClusterList contains a list of IONOSCloudCluster
type IONOSCloudClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IONOSCloudCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IONOSCloudCluster{}, &IONOSCloudClusterList{})
}
