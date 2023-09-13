package context

import (
	"fmt"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/ionos"
	"github.com/go-logr/logr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
)

// MachineContext is a Go context used with a IONOSCloudMachine.
type MachineContext struct {
	*ControllerContext
	Machine           *clusterv1.Machine
	IONOSCloudCluster *v1alpha1.IONOSCloudCluster
	IONOSCloudMachine *v1alpha1.IONOSCloudMachine
	PatchHelper       *patch.Helper
	Logger            logr.Logger
	IONOSClient       ionos.Client
}

// String returns IONOSCloudMachineGroupVersionKind IONOSCloudMachineNamespace/IONOSCloudMachineName.
func (c *MachineContext) String() string {
	return fmt.Sprintf("%s %s/%s", c.IONOSCloudMachine.GroupVersionKind(), c.IONOSCloudMachine.Namespace, c.IONOSCloudMachine.Name)
}

// Patch updates the object and its status on the API server.
func (c *MachineContext) Patch() error {
	return c.PatchHelper.Patch(c, c.IONOSCloudMachine)
}
