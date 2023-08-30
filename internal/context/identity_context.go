package context

import (
	"fmt"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/cluster-api/util/patch"
)

// MachineContext is a Go context used with a IONOSCloudClusterIdentity.
type IdentityContext struct {
	*ControllerContext
	IONOSCloudClusterIdentity *v1alpha1.IONOSCloudClusterIdentity
	PatchHelper               *patch.Helper
	Logger                    logr.Logger
}

// String returns IONOSCloudMachineGroupVersionKind IONOSCloudMachineNamespace/IONOSCloudMachineName.
func (c *IdentityContext) String() string {
	return fmt.Sprintf("%s %s/%s", c.IONOSCloudClusterIdentity.GroupVersionKind(), c.IONOSCloudClusterIdentity.Namespace, c.IONOSCloudClusterIdentity.Name)
}

// Patch updates the object and its status on the API server.
func (c *IdentityContext) Patch() error {
	return c.PatchHelper.Patch(c, c.IONOSCloudClusterIdentity)
}
