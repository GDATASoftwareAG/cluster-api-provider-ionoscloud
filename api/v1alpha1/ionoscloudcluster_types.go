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

// +kubebuilder:validation:Enum=es/vlt;fr/par;de/txl;de/fra;gb-lhr;us-ewr;us-las;
type Location string

func (r Location) String() string {
	return string(r)
}

// IONOSCloudClusterSpec defines the desired state of IONOSCloudCluster
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.dataCenterID) || has(self.dataCenterID)", message="DataCenterID is required once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.loadBalancerID) || has(self.loadBalancerID)", message="LoadBalancerID is required once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.publicLanID) || has(self.publicLanID)", message="PublicLanID is required once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.internetLanID) || has(self.internetLanID)", message="InternetLanID is required once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.privateLanID) || has(self.privateLanID)", message="PrivateLanID is required once set"
type IONOSCloudClusterSpec struct {

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Location is immutable"
	Location Location `json:"location"`

	// +kubebuilder:validation:MinLength=1
	IdentityName string `json:"identityName"`
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
	// +optional
	Lans []IONOSLanSpec `json:"lans,omitempty"`
	// +optional
	LoadBalancer *IONOSLoadBalancerSpec `json:"loadBalancer,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="DataCenterID is immutable"
	DataCenterID string `json:"dataCenterID,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="LoadBalancerID is immutable"
	LoadBalancerID string `json:"loadBalancerID,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="PublicLanID is immutable"
	PublicLanID *int32 `json:"publicLanID,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="InternetLanID is immutable"
	InternetLanID *int32 `json:"internetLanID,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="PrivateLanID is immutable"
	PrivateLanID *int32 `json:"privateLanID,omitempty"`
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

type IONOSLanSpec struct {
	LanID  *int32 `json:"lanID,omitempty"` //validate?
	Name   string `json:"name"`            //validate?
	Public bool   `json:"public"`
	//NameTemplate string   `json:"nameTemplate"`
	//FailoverIPs  []string `json:"failoverIPs,omitempty"`
}

type IONOSLoadBalancerSpec struct {
	ID             string          `json:"id,omitempty"`
	ListenerLanRef IONOSLanRefSpec `json:"listenerLanRef"`
	TargetLanRef   IONOSLanRefSpec `json:"targetLanRef"`
}

func (c *IONOSCloudCluster) Lan(name string) *IONOSLanSpec {
	for i := range c.Spec.Lans {
		if c.Spec.Lans[i].Name == name {
			return &c.Spec.Lans[i]
		}
	}
	return nil
}

func (c *IONOSCloudCluster) LanBy(id *int32) *IONOSLanSpec {
	if id == nil || *id == 0 {
		return nil
	}
	for i := range c.Spec.Lans {
		if *c.Spec.Lans[i].LanID == *id {
			return &c.Spec.Lans[i]
		}
	}
	return nil
}

func (c *IONOSCloudCluster) EnsureLan(spec IONOSLanSpec) {
	if spec.Name == "" {
		return
	}
	for i := range c.Spec.Lans {
		if c.Spec.Lans[i].Name == spec.Name {
			c.Spec.Lans[i] = spec
			return
		}
	}
	c.Spec.Lans = append(c.Spec.Lans, spec)
}
