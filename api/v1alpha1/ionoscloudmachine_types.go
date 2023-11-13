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
	// ServerCreatedCondition documents the creation of the Server
	ServerCreatedCondition clusterv1.ConditionType = "ServerCreated"

	// ServerCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Server.
	ServerCreationFailedReason = "ServerCreationFailed"
)

// IONOSCloudMachineSpec defines the desired state of IONOSCloudMachine
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.providerID) || has(self.providerID)", message="ProviderID is required once set"
type IONOSCloudMachineSpec struct {
	// The name of the  resource.
	Name *string `json:"name,omitempty"`

	// The availability zone in which the server should be provisioned.
	// +kubebuilder:default=AUTO
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="AvailabilityZone is immutable"
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
	// The total number of cores for the enterprise server.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Cores is immutable"
	Cores *int32 `json:"cores"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions; available CPU architectures can be retrieved from the datacenter resource; must not be provided for CUBE servers.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="CpuFamily is immutable"
	CpuFamily *string `json:"cpuFamily"`
	// The memory size for the enterprise server in MB, such as 2048.
	// +kubebuilder:validation:Minimum=256
	// +kubebuilder:validation:MultipleOf=256
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Ram is immutable"
	Ram        *int32          `json:"ram"`
	BootVolume IONOSVolumeSpec `json:"bootVolume"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ProviderID is immutable"
	ProviderID string         `json:"providerID,omitempty"`
	Nics       []IONOSNicSpec `json:"nics,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.sshKeys) || has(self.sshKeys)", message="SSHKeys is required once set"
type IONOSVolumeSpec struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable"
	Type string `json:"type"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Size is immutable"
	Size string `json:"size"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Image is immutable"
	Image string `json:"image"`
	// Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key. This field may only be set in creation requests. When reading, it always returns null. SSH keys are only supported if a public Linux image is used for the volume creation.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="SSHKeys is immutable"
	SSHKeys *[]string `json:"sshKeys,omitempty"`
}

type IONOSNicSpec struct {
	LanRef    IONOSLanRefSpec `json:"lanRef"`
	PrimaryIP *string         `json:"primaryIP,omitempty"`
	//NameTemplate string  `json:"nameTemplate"`
}

// IONOSCloudMachineStatus defines the observed state of IONOSCloudMachine
type IONOSCloudMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`
	// Conditions defines current service state of the IONOSCloudCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
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

func (c *IONOSCloudMachine) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *IONOSCloudMachine) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IONOSCloudMachine{}, &IONOSCloudMachineList{})
}

func (c *IONOSCloudMachine) EnsureNic(spec IONOSNicSpec) {
	if spec.LanRef.Name == "" {
		return
	}
	for i := range c.Spec.Nics {
		if c.Spec.Nics[i].LanRef.Name == spec.LanRef.Name {
			c.Spec.Nics[i].PrimaryIP = spec.PrimaryIP
			c.Spec.Nics[i].LanRef = spec.LanRef
			return
		}
	}
	c.Spec.Nics = append(c.Spec.Nics, spec)
}

func (c *IONOSCloudMachine) NicByLan(name string) *IONOSNicSpec {
	if name == "" {
		return nil
	}
	for i := range c.Spec.Nics {
		if c.Spec.Nics[i].LanRef.Name == name {
			return &c.Spec.Nics[i]
		}
	}
	return nil
}
