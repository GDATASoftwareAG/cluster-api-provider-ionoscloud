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
	"strings"
)

const (
	// ServerCreatedCondition documents the creation of the Server
	ServerCreatedCondition clusterv1.ConditionType = "ServerCreated"

	// ServerCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Server.
	ServerCreationFailedReason = "ServerCreationFailed"

	// VolumeCreatedCondition documents the creation of the Volume
	VolumeCreatedCondition clusterv1.ConditionType = "VolumeCreated"

	// VolumeCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Volume.
	VolumeCreationFailedReason = "VolumeCreationFailed"

	// NicCreatedCondition documents the creation of the Nic
	NicCreatedCondition clusterv1.ConditionType = "NicCreated"

	// NicCreationFailedReason (Severity=Error) documents a controller detecting
	// issues with the creation of the Nic.
	NicCreationFailedReason = "NicCreationFailed"
)

// IONOSCloudMachineSpec defines the desired state of IONOSCloudMachine
type IONOSCloudMachineSpec struct {
	// The availability zone in which the server should be provisioned.
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
	// The total number of cores for the enterprise server.
	Cores *int32 `json:"cores,omitempty"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions; available CPU architectures can be retrieved from the datacenter resource; must not be provided for CUBE servers.
	CpuFamily *string `json:"cpuFamily,omitempty"`
	// The name of the  resource.
	Name *string `json:"name,omitempty"`
	// The placement group ID that belongs to this server; Requires system privileges
	PlacementGroupId *string `json:"placementGroupId,omitempty"`
	// The memory size for the enterprise server in MB, such as 2048. Size must be specified in multiples of 256 MB with a minimum of 256 MB; however, if you set ramHotPlug to TRUE then you must use a minimum of 1024 MB. If you set the RAM size more than 240GB, then ramHotPlug will be set to FALSE and can not be set to TRUE unless RAM size not set to less than 240GB.
	Ram *int32 `json:"ram,omitempty"`

	// +kubebuilder:validation:Required
	BootVolume IONOSVolumeSpec `json:"bootVolume"`

	// primary ip of the virtual machine.
	IP *string `json:"ip,omitempty"`

	ProviderID         string `json:"providerID,omitempty"`
	NetworkInterfaceID string `json:"networkInterfaceID,omitempty"`
	VolumeID           string `json:"volumeID,omitempty"`
}

func (s *IONOSCloudMachineSpec) UnprefixedProviderId() string {
	if strings.HasPrefix(s.ProviderID, "ionos://") {
		return s.ProviderID[8:]
	} else {
		return s.ProviderID
	}
}

type IONOSVolumeSpec struct {
	Type string `json:"type"`
	// +kubebuilder:validation:Required
	Size  string `json:"size"`
	Image string `json:"image"`
	// Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key. This field may only be set in creation requests. When reading, it always returns null. SSH keys are only supported if a public Linux image is used for the volume creation.
	SSHKeys *[]string `json:"sshKeys,omitempty"`
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
