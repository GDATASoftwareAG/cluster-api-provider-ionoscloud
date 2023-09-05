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
	"fmt"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/utils"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const STATE_BUSY = "BUSY"
const STATE_AVAILABLE = "AVAILABLE"

var defaultRetryIntervalOnBusy = time.Minute

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
	logger.Info("Starting Reconcile ionoscloudCluster")

	// Fetch the ionosCloudCluster instance
	ionoscloudCluster := &v1alpha1.IONOSCloudCluster{}
	err := r.K8sClient.Get(ctx, req.NamespacedName, ionoscloudCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("IONOSCloudCluster not found, won't reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := clusterutilv1.GetOwnerCluster(ctx, r.K8sClient, ionoscloudCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		logger.Info("Waiting for Cluster Controller to set OwnerRef on IONOSCloudCluster")
		return reconcile.Result{}, nil
	}

	if annotations.IsPaused(cluster, ionoscloudCluster) {
		logger.Info("IONOSCloudCluster is owned by a cluster that is paused. won't reconcile")
		return reconcile.Result{}, nil
	}

	// Build the patch helper.
	patchHelper, err := patch.NewHelper(ionoscloudCluster, r.K8sClient)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to initialize patch helper")
	}

	user, password, token, host, err := utils.GetLoginDataForCluster(ctx, r.ControllerContext.K8sClient, ionoscloudCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create the cluster context for this request.
	clusterContext := &context.ClusterContext{
		ControllerContext: r.ControllerContext,
		Cluster:           cluster,
		IONOSCloudCluster: ionoscloudCluster,
		Logger:            logger,
		PatchHelper:       patchHelper,
		IONOSClient:       r.IONOSClientFactory.GetClient(user, password, token, host),
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
		return reconcile.Result{}, errors.Wrapf(err, "failed to set owner refs on IONOSCloudMachine objects")
	}

	// Handle deleted clusters
	if !ionoscloudCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(clusterContext)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(clusterContext)
}

func (r *IONOSCloudClusterReconciler) reconcileDelete(ctx *context.ClusterContext) (reconcile.Result, error) {
	ctx.Logger.Info("Deleting IONOSCloudCluster")
	if ctx.IONOSCloudCluster.Spec.DataCenterID != "" {

		resp, err := ctx.IONOSClient.DeleteDatacenter(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID)
		if err != nil && resp.StatusCode != 404 {
			return reconcile.Result{}, err
		}

		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.DataCenterCreatedCondition, "DataCenterDeleted", clusterv1.ConditionSeverityInfo, "")
		ctrlutil.RemoveFinalizer(ctx.IONOSCloudCluster, v1alpha1.ClusterFinalizer)
	}

	return reconcile.Result{}, nil
}

func (r *IONOSCloudClusterReconciler) reconcileNormal(ctx *context.ClusterContext) (reconcile.Result, error) {
	ctx.Logger.Info("Reconciling IONOSCloudCluster")

	// If the IONOSCloudCluster doesn't have our finalizer, add it.
	ctrlutil.AddFinalizer(ctx.IONOSCloudCluster, v1alpha1.ClusterFinalizer)

	if result, err := r.reconcileDataCenter(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.DataCenterCreatedCondition, v1alpha1.DataCenterCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	if result, err := r.reconcilePrivateLan(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.PrivateLanCreatedCondition, v1alpha1.PrivateLanCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	if result, err := r.reconcilePublicLan(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.PublicLanCreatedCondition, v1alpha1.PublicLanCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	if result, err := r.reconcileLoadBalancer(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.LoadBalancerCreatedCondition, v1alpha1.LoadBalancerCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	if result, err := r.reconcileInternet(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.InternetLanCreatedCondition, v1alpha1.InternetLanCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	ctx.IONOSCloudCluster.Status.Ready = true

	return reconcile.Result{}, nil
}

func (r *IONOSCloudClusterReconciler) reconcileDataCenter(ctx *context.ClusterContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling DataCenter")
	if ctx.IONOSCloudCluster.Spec.DataCenterID == "" {
		datacenterName := fmt.Sprintf("cluster-%s", ctx.Cluster.Name)
		datacenter, _, err := ctx.IONOSClient.CreateDatacenter(ctx, datacenterName, ctx.IONOSCloudCluster.Spec.Location)

		if err != nil {
			return &reconcile.Result{}, errors.Wrapf(err, "error creating datacenter %v", datacenter)
		}

		ctx.IONOSCloudCluster.Spec.DataCenterID = *datacenter.Id
	}

	// check status
	datacenter, resp, err := ctx.IONOSClient.GetDatacenter(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID)
	if err != nil && resp.StatusCode != 404 {
		return &reconcile.Result{}, errors.Wrap(err, "error getting private Lan")
	}

	if resp.StatusCode == 404 || *datacenter.Metadata.State == STATE_BUSY {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.Wrap(err, "datacenter not available (yet)")
	}

	conditions.MarkTrue(ctx.IONOSCloudCluster, v1alpha1.DataCenterCreatedCondition)

	return nil, nil
}

func (r *IONOSCloudClusterReconciler) reconcilePrivateLan(ctx *context.ClusterContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling private Lan")
	if ctx.IONOSCloudCluster.Spec.PrivateLanID == nil {
		lanID, err := createLan(ctx, false)
		if err != nil {
			return &reconcile.Result{}, errors.Wrap(err, "error creating private Lan")
		}
		ctx.IONOSCloudCluster.Spec.PrivateLanID = lanID
	}

	// check status
	lanId := fmt.Sprint(*ctx.IONOSCloudCluster.Spec.PrivateLanID)
	lan, resp, err := ctx.IONOSClient.GetLan(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lanId)

	if err != nil && resp.StatusCode != 404 {
		return &reconcile.Result{}, errors.Wrap(err, "error getting private Lan")
	}

	if resp.StatusCode == 404 || *lan.Metadata.State == STATE_BUSY {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.Wrap(err, "private Lan not available (yet)")
	}

	conditions.MarkTrue(ctx.IONOSCloudCluster, v1alpha1.PrivateLanCreatedCondition)

	return nil, nil
}

func (r *IONOSCloudClusterReconciler) reconcilePublicLan(ctx *context.ClusterContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling public Lan")
	if ctx.IONOSCloudCluster.Spec.PublicLanID == nil {
		lanID, err := createLan(ctx, true)
		if err != nil {
			return &reconcile.Result{}, err
		}
		ctx.IONOSCloudCluster.Spec.PublicLanID = lanID
	}

	// check status
	lanId := fmt.Sprint(*ctx.IONOSCloudCluster.Spec.PublicLanID)
	lan, resp, err := ctx.IONOSClient.GetLan(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lanId)

	if err != nil && resp.StatusCode != 404 {
		return &reconcile.Result{}, errors.Wrap(err, "error getting public Lan")
	}

	if resp.StatusCode == 404 || *lan.Metadata.State == STATE_BUSY {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.Wrap(err, "public Lan not available (yet)")
	}

	conditions.MarkTrue(ctx.IONOSCloudCluster, v1alpha1.PublicLanCreatedCondition)
	return nil, nil
}

func (r *IONOSCloudClusterReconciler) reconcileInternet(ctx *context.ClusterContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling internet")
	if ctx.IONOSCloudCluster.Spec.InternetLanID == nil {
		lanID, err := createLan(ctx, true)
		if err != nil {
			return &reconcile.Result{}, err
		}
		ctx.IONOSCloudCluster.Spec.InternetLanID = lanID
	}

	// check status
	lanId := fmt.Sprint(*ctx.IONOSCloudCluster.Spec.InternetLanID)
	lan, resp, err := ctx.IONOSClient.GetLan(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lanId)

	if err != nil && resp.StatusCode != 404 {
		return &reconcile.Result{}, errors.Wrap(err, "error getting internet Lan")
	}

	if resp.StatusCode == 404 || *lan.Metadata.State == STATE_BUSY {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.Wrap(err, "internet Lan not available (yet)")
	}

	conditions.MarkTrue(ctx.IONOSCloudCluster, v1alpha1.InternetLanCreatedCondition)

	return nil, nil
}

func (r *IONOSCloudClusterReconciler) reconcileLoadBalancer(ctx *context.ClusterContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling LoadBalancer")
	if ctx.IONOSCloudCluster.Spec.LoadBalancerID == "" {
		loadBalancerName := fmt.Sprintf("lb-%s", ctx.Cluster.Name)
		loadBalancer := ionoscloud.NetworkLoadBalancer{
			Entities: &ionoscloud.NetworkLoadBalancerEntities{
				Forwardingrules: &ionoscloud.NetworkLoadBalancerForwardingRules{
					Items: &[]ionoscloud.NetworkLoadBalancerForwardingRule{
						{
							Properties: &ionoscloud.NetworkLoadBalancerForwardingRuleProperties{
								ListenerIp:   ionoscloud.ToPtr(ctx.IONOSCloudCluster.Spec.ControlPlaneEndpoint.Host),
								ListenerPort: ionoscloud.PtrInt32(6443),
								Algorithm:    ionoscloud.ToPtr("ROUND_ROBIN"),
								Name:         ionoscloud.ToPtr(ctx.IONOSCloudCluster.Name),
								Protocol:     ionoscloud.ToPtr("TCP"),
							},
						},
					},
				},
			},
			Properties: &ionoscloud.NetworkLoadBalancerProperties{
				ListenerLan: ctx.IONOSCloudCluster.Spec.PublicLanID,
				Name:        &loadBalancerName,
				TargetLan:   ctx.IONOSCloudCluster.Spec.PrivateLanID,
				Ips:         &[]string{ctx.IONOSCloudCluster.Spec.ControlPlaneEndpoint.Host},
			},
		}
		loadBalancer, _, err := ctx.IONOSClient.CreateLoadBalancer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, loadBalancer)
		if err != nil {
			return &reconcile.Result{}, err
		}
		ctx.IONOSCloudCluster.Spec.LoadBalancerID = *loadBalancer.Id
	}

	// check status
	loadBalancer, resp, err := ctx.IONOSClient.GetLoadBalancer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudCluster.Spec.LoadBalancerID)
	if err != nil && resp.StatusCode != 404 {
		return &reconcile.Result{}, errors.Wrap(err, "error getting loadbalancer")
	}

	if resp.StatusCode == 404 || *loadBalancer.Metadata.State == STATE_BUSY {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.New("loadbalancer not available (yet)")
	}

	conditions.MarkTrue(ctx.IONOSCloudCluster, v1alpha1.LoadBalancerCreatedCondition)

	return nil, nil
}

func createLan(ctx *context.ClusterContext, public bool) (*int32, error) {
	lan, _, err := ctx.IONOSClient.CreateLan(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, public)
	if err != nil {
		return nil, err
	}

	if id, err := utils.ToInt32(*lan.Id); err != nil {
		return nil, err
	} else {
		return &id, nil
	}
}

func setOwnerRefsOnIONOSCloudMachines(ctx *context.ClusterContext) error {
	ionoscloudMachines, err := utils.GetIONOSCloudMachinesInCluster(ctx, ctx.K8sClient, ctx.Cluster.Namespace, ctx.Cluster.Name)
	if err != nil {
		return errors.Wrapf(err,
			"unable to list IONOSCloudMachines part of IONOSCloudCluster %s/%s", ctx.IONOSCloudCluster.Namespace, ctx.IONOSCloudCluster.Name)
	}

	var patchErrors []error
	for _, ionoscloudMachine := range ionoscloudMachines {
		patchHelper, err := patch.NewHelper(ionoscloudMachine, ctx.K8sClient)
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
		For(&v1alpha1.IONOSCloudCluster{}).
		// Watch the CAPI resource that owns this infrastructure resource.
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterutilv1.ClusterToInfrastructureMapFunc(
				r.Context,
				schema.GroupVersionKind{},
				mgr.GetClient(), &v1alpha1.IONOSCloudCluster{},
			)),
		).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(r.Logger)).
		Complete(r)
}
