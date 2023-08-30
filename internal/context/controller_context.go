package context

import (
	"context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/ionos"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	context.Context

	// K8sClient is the controller manager's client.
	K8sClient client.Client

	// Logger is the controller manager's logger.
	Logger logr.Logger

	// Scheme is the controller manager's API scheme.
	Scheme *runtime.Scheme

	// IONOSClientFactory is the controller manager's factory for IONOSClients
	IONOSClientFactory *ionos.ClientFactory
}
