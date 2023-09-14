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
	b64 "encoding/base64"
	"fmt"
	v1alpha1 "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/utils"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	apitypes "k8s.io/apimachinery/pkg/types"
	"net/http"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

// IONOSCloudMachineReconciler reconciles a IONOSCloudMachine object
type IONOSCloudMachineReconciler struct {
	*context.ControllerContext
}

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *IONOSCloudMachineReconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Logger.WithName(req.Namespace).WithName(req.Name)
	logger.Info("Starting Reconcile ionoscloudMachine")

	// Fetch the ionosCloudMachine instance
	ionoscloudMachine := &v1alpha1.IONOSCloudMachine{}
	if err := r.K8sClient.Get(ctx, req.NamespacedName, ionoscloudMachine); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("IONOSCloudMachine not found, won't reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Machine.
	machine, err := clusterutilv1.GetOwnerMachine(ctx, r.K8sClient, ionoscloudMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on IONOSCloudMachine")
		return reconcile.Result{}, nil
	}

	if machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Waiting for the Bootstrap provider to set the datasecret")
		return reconcile.Result{}, nil
	}

	// Fetch the Cluster.
	cluster := &clusterv1.Cluster{}
	if err := r.K8sClient.Get(ctx, types.NamespacedName{Namespace: machine.Namespace, Name: machine.Spec.ClusterName}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Cluster not found, won't reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if annotations.IsPaused(cluster, ionoscloudMachine) {
		logger.Info("IONOSCloudMachine %s/%s linked to a cluster that is paused",
			ionoscloudMachine.Namespace, ionoscloudMachine.Name)
		return reconcile.Result{}, nil
	}

	// Fetch the ionosCloudCluster instance
	ionoscloudCluster := &v1alpha1.IONOSCloudCluster{}
	if err := r.K8sClient.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: cluster.Spec.InfrastructureRef.Name}, ionoscloudCluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("IONOSCloudCluster not found, won't reconcile")
			return reconcile.Result{}, nil
		}

		if ionoscloudCluster.Spec.DataCenterID == "" {
			return reconcile.Result{}, errors.Wrapf(err, "invalid datacenterid")
		}
		return reconcile.Result{}, err
	}

	// Build the patch helper.
	patchHelper, err := patch.NewHelper(ionoscloudMachine, r.K8sClient)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to initialize patch helper")
	}

	user, password, token, host, err := utils.GetLoginDataForCluster(ctx, r.ControllerContext.K8sClient, ionoscloudCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create the cluster context for this request.
	machineContext := &context.MachineContext{
		ControllerContext: r.ControllerContext,
		Machine:           machine,
		IONOSCloudCluster: ionoscloudCluster,
		IONOSCloudMachine: ionoscloudMachine,
		Logger:            logger,
		PatchHelper:       patchHelper,
		IONOSClient:       r.IONOSClientFactory.GetClient(user, password, token, host),
	}

	// Always issue a patch when exiting this function so changes to the
	// resource are patched back to the API server.
	defer func() {
		if err := machineContext.Patch(); err != nil {
			if reterr == nil {
				reterr = err
			}
			machineContext.Logger.Error(err, "patch failed", "cluster", machineContext.String())
		}
	}()

	// Handle deleted clusters
	if !ionoscloudMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(machineContext)
	}
	// Handle non-deleted clusters
	return r.reconcileNormal(machineContext)
}

func (r *IONOSCloudMachineReconciler) reconcileDelete(ctx *context.MachineContext) (reconcile.Result, error) {
	ctx.Logger.Info("Deleting IONOSCloudMachine")
	if ctx.IONOSCloudMachine.Spec.ProviderID != "" {
		server, resp, err := ctx.IONOSClient.GetServer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudMachine.Spec.ProviderID)
		if err != nil && resp.StatusCode != http.StatusNotFound {
			return reconcile.Result{}, err
		}

		if resp.StatusCode != http.StatusNotFound {
			_, err = ctx.IONOSClient.DeleteServer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudMachine.Spec.ProviderID)
			if err != nil {
				return reconcile.Result{}, err
			}

			for _, volume := range *server.Entities.Volumes.Items {
				_, err = ctx.IONOSClient.DeleteVolume(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, *volume.Id)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}

		if ctx.IONOSCloudMachine.Spec.IP != nil {
			if rules, _, err := ctx.IONOSClient.GetLoadBalancerForwardingRules(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudCluster.Spec.LoadBalancerID); err != nil {
				return reconcile.Result{}, err
			} else {
				targetToDelete := ionoscloud.NetworkLoadBalancerForwardingRuleTarget{
					Ip:     ctx.IONOSCloudMachine.Spec.IP,
					Port:   ionoscloud.PtrInt32(6443),
					Weight: ionoscloud.PtrInt32(1),
				}
				for _, rule := range *rules.Items {
					if *rule.Properties.ListenerPort != 6443 {
						// we only care about the api server port
						continue
					}

					for _, target := range *rule.Properties.Targets {
						if *target.Ip == *targetToDelete.Ip {
							targets := *rule.Properties.Targets
							targets = findAndDeleteByIP(targets, targetToDelete)
							properties := ionoscloud.NetworkLoadBalancerForwardingRuleProperties{
								Algorithm:    rule.Properties.Algorithm,
								HealthCheck:  rule.Properties.HealthCheck,
								ListenerIp:   rule.Properties.ListenerIp,
								ListenerPort: rule.Properties.ListenerPort,
								Name:         rule.Properties.Name,
								Protocol:     rule.Properties.Protocol,
								Targets:      &targets,
							}
							if _, _, err = ctx.IONOSClient.PatchLoadBalancerForwardingRule(
								ctx,
								ctx.IONOSCloudCluster.Spec.DataCenterID,
								ctx.IONOSCloudCluster.Spec.LoadBalancerID,
								*rule.Id,
								properties,
							); err != nil {
								return reconcile.Result{}, err
							}

						}
					}
				}
			}
		}

		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.ServerCreatedCondition, "ServerDeleted", clusterv1.ConditionSeverityInfo, "")
		ctrlutil.RemoveFinalizer(ctx.IONOSCloudMachine, v1alpha1.MachineFinalizer)
	}

	return reconcile.Result{}, nil
}

func (r *IONOSCloudMachineReconciler) reconcileNormal(ctx *context.MachineContext) (reconcile.Result, error) {
	ctx.Logger.Info("Reconciling IONOSCloudCluster")

	// If the IONOSCloudMachine doesn't have our finalizer, add it.
	ctrlutil.AddFinalizer(ctx.IONOSCloudMachine, v1alpha1.MachineFinalizer)

	if result, err := r.reconcileServer(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.ServerCreatedCondition, v1alpha1.ServerCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	if result, err := r.reconcileLoadBalancerForwardingRule(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.LoadBalancerForwardingRuleCreatedCondition, v1alpha1.LoadBalancerForwardingRuleCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, err
	}

	server, _, err := ctx.IONOSClient.GetServer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudMachine.Spec.ProviderID)

	if err != nil {
		return reconcile.Result{}, err
	}

	if *server.Metadata.State == STATE_AVAILABLE {
		ctx.IONOSCloudMachine.Status.Ready = true
	} else {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Minute,
		}, nil
	}

	return reconcile.Result{}, nil
}

// getBootstrapData gets a machine's bootstrap data from the corresponding k8s secret and returns an error if
// the format is not cloud-init.
func (r *IONOSCloudMachineReconciler) getBootstrapData(ctx *context.MachineContext) (string, error) {
	if ctx.Machine.Spec.Bootstrap.DataSecretName == nil {
		ctx.Logger.Info("Machine has no bootstrap data")
		return "", nil
	}

	secret := &corev1.Secret{}
	secretKey := apitypes.NamespacedName{
		Namespace: ctx.Machine.Namespace,
		Name:      *ctx.Machine.Spec.Bootstrap.DataSecretName,
	}
	if err := ctx.K8sClient.Get(ctx, secretKey, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for %s", ctx)
	}

	format, ok := secret.Data["format"]
	if !ok || string(format) != string(bootstrapv1.CloudConfig) {
		return "", errors.New("error retrieving bootstrap data: only format cloud-config is currently supported")
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	replaced := strings.Replace(string(value), "{{ ds.meta_data.hostname }}", ctx.IONOSCloudMachine.Name, -1)
	replaced = strings.Replace(replaced, "{{ ds.meta_data.datacenter_id }}", ctx.IONOSCloudCluster.Spec.DataCenterID, -1)

	userdata := b64.StdEncoding.EncodeToString([]byte(replaced))

	return userdata, nil
}

func (r *IONOSCloudMachineReconciler) reconcileServer(ctx *context.MachineContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling server")
	if ctx.IONOSCloudMachine.Spec.ProviderID == "" {
		// Get the bootstrap data.
		bootstrapData, err := r.getBootstrapData(ctx)
		if err != nil {
			return &reconcile.Result{}, errors.Wrapf(err, "unable to get bootstrap data")
		}

		diskSize, err := utils.ToFloat32(ctx.IONOSCloudMachine.Spec.BootVolume.Size)
		if err != nil {
			return &reconcile.Result{}, errors.Wrapf(err, "invalid spec.bootvolume.disksize")
		}

		server := ionoscloud.Server{
			Entities: &ionoscloud.ServerEntities{
				Nics: &ionoscloud.Nics{
					Items: &[]ionoscloud.Nic{
						{
							Properties: &ionoscloud.NicProperties{
								Dhcp: ionoscloud.ToPtr(true),
								Lan:  ctx.IONOSCloudCluster.Spec.PrivateLanID,
								Name: ionoscloud.ToPtr(fmt.Sprintf("%s-nic-lb", ctx.IONOSCloudMachine.Name)),
							},
						},
						{
							Properties: &ionoscloud.NicProperties{
								Dhcp: ionoscloud.ToPtr(true),
								Lan:  ctx.IONOSCloudCluster.Spec.InternetLanID,
								Name: ionoscloud.ToPtr(fmt.Sprintf("%s-nic-internet", ctx.IONOSCloudMachine.Name)),
							},
						},
					},
				},
				Volumes: &ionoscloud.AttachedVolumes{
					Items: &[]ionoscloud.Volume{
						{
							Properties: &ionoscloud.VolumeProperties{
								DiscVirtioHotPlug:   ionoscloud.ToPtr(true),
								DiscVirtioHotUnplug: ionoscloud.ToPtr(true),
								NicHotPlug:          ionoscloud.ToPtr(true),
								NicHotUnplug:        ionoscloud.ToPtr(true),
								Image:               ionoscloud.ToPtr(ctx.IONOSCloudMachine.Spec.BootVolume.Image),
								Name:                ionoscloud.ToPtr(fmt.Sprintf("%s-storage", ctx.IONOSCloudMachine.Name)),
								Size:                &diskSize,
								Type:                ionoscloud.ToPtr(ctx.IONOSCloudMachine.Spec.BootVolume.Type),
								UserData:            &bootstrapData,
							},
						},
					},
				},
			},
			Properties: &ionoscloud.ServerProperties{

				Cores:     ctx.IONOSCloudMachine.Spec.Cores,
				CpuFamily: ctx.IONOSCloudMachine.Spec.CpuFamily,
				Name:      ionoscloud.ToPtr(ctx.IONOSCloudMachine.Name),
				Ram:       ctx.IONOSCloudMachine.Spec.Ram,
				Type:      ionoscloud.ToPtr("ENTERPRISE"),
			},
		}
		server, _, err = ctx.IONOSClient.CreateServer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, server)
		if err != nil {
			return &reconcile.Result{}, errors.Wrapf(err, "error creating server %v", server)
		}
		ctx.IONOSCloudMachine.Spec.ProviderID = *server.Id
	}

	// check status
	server, resp, err := ctx.IONOSClient.GetServer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudMachine.Spec.ProviderID)

	if err != nil && resp.StatusCode != http.StatusNotFound {
		return &reconcile.Result{}, errors.Wrap(err, "error getting server")
	}

	if resp.StatusCode == http.StatusNotFound || (server.Metadata != nil && *server.Metadata.State == STATE_BUSY) {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.New("server not yet created")
	}

	nics := *server.Entities.Nics.Items
	for _, nic := range nics {
		if strings.HasSuffix(*nic.Properties.Name, "-nic-lb") {
			ips := *nic.Properties.Ips
			ctx.IONOSCloudMachine.Spec.IP = &ips[0]
		}
	}

	if ctx.IONOSCloudMachine.Spec.IP == nil {
		return &reconcile.Result{RequeueAfter: defaultRetryIntervalOnBusy}, errors.New("server does not have an ip yet")
	}

	conditions.MarkTrue(ctx.IONOSCloudMachine, v1alpha1.ServerCreatedCondition)

	return nil, nil
}

func (r *IONOSCloudMachineReconciler) reconcileLoadBalancerForwardingRule(ctx *context.MachineContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling load balancer forwarding rule")

	if !clusterutilv1.IsControlPlaneMachine(ctx.Machine) {
		ctx.Logger.Info("Reconciled IONOSCLoudMachine is not a control plane...no forwarding rule needed.")
		return nil, nil
	}

	if ctx.IONOSCloudMachine.Spec.IP == nil {
		return &reconcile.Result{}, errors.New("server has not obtained an ip yet")
	}

	if ctx.IONOSCloudMachine.Spec.ProviderID != "" {
		rules, _, err := ctx.IONOSClient.GetLoadBalancerForwardingRules(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudCluster.Spec.LoadBalancerID)
		if err != nil {
			return &reconcile.Result{}, errors.Wrap(err, "error getting forwarding rules")
		}
		desiredTarget := ionoscloud.NetworkLoadBalancerForwardingRuleTarget{
			Ip:     ctx.IONOSCloudMachine.Spec.IP,
			Port:   ionoscloud.PtrInt32(6443),
			Weight: ionoscloud.PtrInt32(1),
		}
		found := false
		for _, rule := range *rules.Items {
			if *rule.Properties.ListenerPort != 6443 {
				// we only care about the api server port
				continue
			}

			if rule.Properties.Targets == nil {
				rule.Properties.Targets = &[]ionoscloud.NetworkLoadBalancerForwardingRuleTarget{}
			}

			for _, target := range *rule.Properties.Targets {
				if *target.Ip == *desiredTarget.Ip {
					found = true
				}
			}

			if !found {
				targets := *rule.Properties.Targets
				targets = append(targets, desiredTarget)
				properties := ionoscloud.NetworkLoadBalancerForwardingRuleProperties{
					Algorithm:    rule.Properties.Algorithm,
					HealthCheck:  rule.Properties.HealthCheck,
					ListenerIp:   rule.Properties.ListenerIp,
					ListenerPort: rule.Properties.ListenerPort,
					Name:         rule.Properties.Name,
					Protocol:     rule.Properties.Protocol,
					Targets:      &targets,
				}

				if _, _, err = ctx.IONOSClient.PatchLoadBalancerForwardingRule(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudCluster.Spec.LoadBalancerID, *rule.Id, properties); err != nil {
					return &reconcile.Result{}, errors.Wrap(err, "error updating forwarding rules")
				}
			}
		}

		conditions.MarkTrue(ctx.IONOSCloudMachine, v1alpha1.LoadBalancerForwardingRuleCreatedCondition)
	}
	return nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IONOSCloudMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&clusterv1.Machine{}, handler.EnqueueRequestsFromMapFunc(clusterutilv1.MachineToInfrastructureMapFunc(v1alpha1.GroupVersion.WithKind("IONOSCloudMachine")))).
		For(&v1alpha1.IONOSCloudMachine{}).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(r.Logger)).
		Complete(r)
}

func findAndDeleteByIP(s []ionoscloud.NetworkLoadBalancerForwardingRuleTarget, item ionoscloud.NetworkLoadBalancerForwardingRuleTarget) []ionoscloud.NetworkLoadBalancerForwardingRuleTarget {
	index := 0
	for _, i := range s {
		if *i.Ip != *item.Ip {
			s[index] = i
			index++
		}
	}
	return s[:index]
}
