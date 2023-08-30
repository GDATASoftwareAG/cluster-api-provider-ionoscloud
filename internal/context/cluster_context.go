package context

import (
	"fmt"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/ionos"
	"github.com/go-logr/logr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
)

// ClusterContext is a Go context used with a IONOSCloudCluster.
type ClusterContext struct {
	*ControllerContext
	Cluster           *clusterv1.Cluster
	IONOSCloudCluster *v1alpha1.IONOSCloudCluster
	PatchHelper       *patch.Helper
	Logger            logr.Logger
	IONOSClient       ionos.IONOSClient
}

// String returns IONOSCloudClusterGroupVersionKind IONOSCloudClusterNamespace/IONOSCloudClusterName.
func (c *ClusterContext) String() string {
	return fmt.Sprintf("%s %s/%s", c.IONOSCloudCluster.GroupVersionKind(), c.IONOSCloudCluster.Namespace, c.IONOSCloudCluster.Name)
}

// Patch updates the object and its status on the API server.
func (c *ClusterContext) Patch() error {
	return c.PatchHelper.Patch(c, c.IONOSCloudCluster)
}
