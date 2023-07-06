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

package controller

import (
	goctx "context"
	infrastructurev1alpha1 "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/pkg/context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/pkg/util"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// IONOSCloudClusterReconciler reconciles a IONOSCloudCluster object
type IONOSCloudClusterReconciler struct {
	*context.ControllerContext
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters/finalizers,verbs=update

// Reconcile ensures the back-end state reflects the Kubernetes resource state intent.
func (r *IONOSCloudClusterReconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Logger.WithName(req.Namespace).WithName(req.Name)
	logger.V(3).Info("Starting Reconcile ionoscloudCluster")

	// Fetch the ionosCloudCluster instance
	ionoscloudCluster := &infrastructurev1alpha1.IONOSCloudCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, ionoscloudCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.V(4).Info("IONOSCloudCluster not found, won't reconcile", "key", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := clusterutilv1.GetOwnerCluster(ctx, r.Client, ionoscloudCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Build the patch helper.
	patchHelper, err := patch.NewHelper(ionoscloudCluster, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to initialize patch helper")
	}

	// Create the cluster context for this request.
	clusterContext := &context.ClusterContext{
		ControllerContext: r.ControllerContext,
		Cluster:           cluster,
		IONOSCloudCluster: ionoscloudCluster,
		Logger:            r.Logger.WithName(req.Namespace).WithName(req.Name),
		PatchHelper:       patchHelper,
	}

	// Always issue a patch when exiting this function so changes to the
	// resource are patched back to the API server.
	defer func() {
		if err := clusterContext.Patch(); err != nil {
			if reterr == nil {
				reterr = err
			}
			clusterContext.Logger.Error(err, "patch failed", "cluster", clusterContext.String())
		}
	}()

	if err := setOwnerRefsOnIONOSCloudMachines(clusterContext); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to set owner refs on VSphereMachine objects")
	}

	// Handle deleted clusters
	if !ionoscloudCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(clusterContext)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(clusterContext)
}

func (r *IONOSCloudClusterReconciler) reconcileDelete(ctx *context.ClusterContext) (reconcile.Result, error) {
	// TODO
	return reconcile.Result{}, nil
}

func (r *IONOSCloudClusterReconciler) reconcileNormal(ctx *context.ClusterContext) (reconcile.Result, error) {
	// TODO
	return reconcile.Result{}, nil
}

func setOwnerRefsOnIONOSCloudMachines(ctx *context.ClusterContext) error {
	ionoscloudMachines, err := util.GetIONOSCloudMachinesInCluster(ctx, ctx.Client, ctx.Cluster.Namespace, ctx.Cluster.Name)
	if err != nil {
		return errors.Wrapf(err,
			"unable to list IONOSCloudMachines part of IONOSCloudCluster %s/%s", ctx.IONOSCloudCluster.Namespace, ctx.IONOSCloudCluster.Name)
	}

	var patchErrors []error
	for _, ionoscloudMachine := range ionoscloudMachines {
		patchHelper, err := patch.NewHelper(ionoscloudMachine, ctx.Client)
		if err != nil {
			patchErrors = append(patchErrors, err)
			continue
		}

		ionoscloudMachine.SetOwnerReferences(clusterutilv1.EnsureOwnerRef(
			ionoscloudMachine.OwnerReferences,
			metav1.OwnerReference{
				APIVersion: ctx.IONOSCloudCluster.APIVersion,
				Kind:       ctx.IONOSCloudCluster.Kind,
				Name:       ctx.IONOSCloudCluster.Name,
				UID:        ctx.IONOSCloudCluster.UID,
			}))

		if err := patchHelper.Patch(ctx, ionoscloudMachine); err != nil {
			patchErrors = append(patchErrors, err)
		}
	}
	return kerrors.NewAggregate(patchErrors)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IONOSCloudClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.IONOSCloudCluster{}).
		Complete(r)
}
