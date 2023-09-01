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
	// ClusterFinalizer allows ReconcileIONOSCloudCluster to clean up ionos cloud
	// resources associated with IONOSCloudCluster before removing it from the
	// API server.
	ClusterFinalizer = "ionoscloudcluster.infrastructure.cluster.x-k8s.io/finalizer"

	// MachineFinalizer allows ReconcileIONOSCloudMachine to clean up ionos cloud
	// resources associated with IONOSCloudMachine before removing it from the
	// API server.
	MachineFinalizer = "ionoscloudmachine.infrastructure.cluster.x-k8s.io/finalizer"

	// IdentityFinalizer allows ReconcileIONOSCloudClusterIdentity to clean up ionos cloud
	// resources associated with IONOSCloudClusterIdentity before removing it from the
	// API server.
	IdentityFinalizer = "ionoscloudclusteridentity.infrastructure.cluster.x-k8s.io/finalizer"

	// LoadBalancerCreatedCondition documents the creation of the loadbalancer
	LoadBalancerCreatedCondition clusterv1.ConditionType = "LoadBalancerCreated"

	// LoadBalancerCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the loadbalancer.
	LoadBalancerCreationFailedReason = "LoadBalancerCreationFailed"

	// DataCenterCreatedCondition documents the creation of the datacenter
	DataCenterCreatedCondition clusterv1.ConditionType = "DataCenterCreated"

	// DataCenterCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the datacenter.
	DataCenterCreationFailedReason = "DataCenterCreationFailed"

	// PublicLanCreatedCondition documents the creation of the Lan
	PublicLanCreatedCondition clusterv1.ConditionType = "PublicLanCreated"

	// PublicLanCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Lan.
	PublicLanCreationFailedReason = "PublicLanCreationFailed"

	// PrivateLanCreatedCondition documents the creation of the Lan
	PrivateLanCreatedCondition clusterv1.ConditionType = "PrivateLanCreated"

	// PrivateLanCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Lan.
	PrivateLanCreationFailedReason = "PrivateLanCreationFailed"

	// InternetLanCreatedCondition documents the creation of the Lan
	InternetLanCreatedCondition clusterv1.ConditionType = "InternetLanCreated"

	// InternetLanCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Lan.
	InternetLanCreationFailedReason = "InternetLanCreationFailed"

	// LoadBalancerForwardingRuleCreatedCondition documents the creation of the ForwardingRule
	LoadBalancerForwardingRuleCreatedCondition clusterv1.ConditionType = "LoadBalancerForwardingRuleCreated"

	// LoadBalancerForwardingRuleCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the ForwardingRule.
	LoadBalancerForwardingRuleCreationFailedReason = "LoadBalancerForwardingRuleCreationFailed"
)

// +kubebuilder:validation:Enum=de/txl;de/fra
type Location string

// IONOSCloudClusterSpec defines the desired state of IONOSCloudCluster
type IONOSCloudClusterSpec struct {
	Location             string                `json:"location,omitempty"` // TODO: make immutable, see https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html
	IdentityName         string                `json:"identityName,omitempty"`
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	DataCenterID   string `json:"dataCenterID,omitempty"`
	LoadBalancerID string `json:"loadBalancerID,omitempty"`
	PublicLanID    *int32 `json:"publicLanID,omitempty"`
	InternetLanID  *int32 `json:"internetLanID,omitempty"`
	PrivateLanID   *int32 `json:"privateLanID,omitempty"`
}

// IONOSCloudClusterStatus defines the observed state of IONOSCloudCluster
type IONOSCloudClusterStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`
	// Conditions defines current service state of the IONOSCloudCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
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

func (c *IONOSCloudCluster) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *IONOSCloudCluster) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IONOSCloudCluster{}, &IONOSCloudClusterList{})
}
