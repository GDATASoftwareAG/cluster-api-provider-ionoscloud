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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	CredentialsAvailableCondition clusterv1.ConditionType = "CredentialsAvailable"
	SecretNotAvailableReason                              = "SecretNotAvailable"

	CredentialsValidCondition clusterv1.ConditionType = "CredentialsValid"
	CredentialsInvalidReason                          = "CredentialsInvalid"
)

// IONOSCloudClusterIdentitySpec defines the desired state of IONOSCloudClusterIdentity
type IONOSCloudClusterIdentitySpec struct {
	// SecretName references a Secret inside the controller namespace with the credentials to use
	// +kubebuilder:validation:MinLength=1
	SecretName string `json:"secretName,omitempty"`
	HostUrl    string `json:"hostUrl"`
}

// IONOSCloudClusterIdentityStatus defines the observed state of IONOSCloudClusterIdentity
type IONOSCloudClusterIdentityStatus struct {
	// Ready is true when the resource is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IONOSCloudCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IONOSCloudClusterIdentity is the Schema for the ionoscloudidentities API
type IONOSCloudClusterIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IONOSCloudClusterIdentitySpec   `json:"spec,omitempty"`
	Status IONOSCloudClusterIdentityStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IONOSCloudClusterIdentityList contains a list of IONOSCloudClusterIdentity
type IONOSCloudClusterIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IONOSCloudClusterIdentity `json:"items"`
}

func (c *IONOSCloudClusterIdentity) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *IONOSCloudClusterIdentity) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IONOSCloudClusterIdentity{}, &IONOSCloudClusterIdentityList{})
}
