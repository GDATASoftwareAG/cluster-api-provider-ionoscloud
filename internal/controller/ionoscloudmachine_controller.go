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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/ionos"

	"go.uber.org/multierr"

	v1alpha1 "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/utils"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	apitypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const csiVolumePrefix = "csi-pv."

var defaultMachineRetryIntervalOnBusy = time.Second * 30

// IONOSCloudMachineReconciler reconciles a IONOSCloudMachine object
type IONOSCloudMachineReconciler struct {
	*context.ControllerContext
}

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/finalizers,verbs=update
//+kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *IONOSCloudMachineReconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Logger.WithName(req.Namespace).WithName(req.Name)
	logger.Info("Starting Reconcile IONOSCloudMachine")

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
				if strings.HasPrefix(*volume.Properties.Name, csiVolumePrefix) { //ignore csi volumes
					continue
				}
				_, err = ctx.IONOSClient.DeleteVolume(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, *volume.Id)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}

		// ensure server is deleted
		_, resp, err = ctx.IONOSClient.GetServer(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudMachine.Spec.ProviderID)
		if resp.StatusCode == http.StatusOK {
			return reconcile.Result{RequeueAfter: defaultMachineRetryIntervalOnBusy}, nil
		} else if resp.StatusCode != http.StatusNotFound {
			return reconcile.Result{}, err
		}

		if err = r.reconcileDeleteLoadBalancerForwardingRule(ctx); err != nil {
			return reconcile.Result{}, err
		}

		conditions.MarkFalse(ctx.IONOSCloudCluster, v1alpha1.ServerCreatedCondition, "ServerDeleted", clusterv1.ConditionSeverityInfo, "")
	}

	// Remove finalizers from any ipam claims
	for devIdx, device := range ctx.IONOSCloudMachine.Spec.Nics {
		if device.PrimaryAddressFrom == nil {
			continue
		}

		// check if claim exists
		ipAddrClaim := &ipamv1.IPAddressClaim{}
		ipAddrClaimName := IPAddressClaimName(ctx.IONOSCloudMachine.Name, devIdx, 0)
		ctx.Logger.Info("removing finalizer", "IPAddressClaim", ipAddrClaimName)
		ipAddrClaimKey := apitypes.NamespacedName{
			Namespace: ctx.IONOSCloudMachine.Namespace,
			Name:      ipAddrClaimName,
		}
		if err := ctx.K8sClient.Get(ctx, ipAddrClaimKey, ipAddrClaim); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return reconcile.Result{}, errors.Wrapf(err, fmt.Sprintf("failed to find IPAddressClaim %q to remove the finalizer", ipAddrClaimName))
		}
		if ctrlutil.RemoveFinalizer(ipAddrClaim, v1alpha1.MachineFinalizer) {
			if err := ctx.K8sClient.Update(ctx, ipAddrClaim); err != nil {
				return reconcile.Result{}, errors.Wrapf(err, fmt.Sprintf("failed to update IPAddressClaim %q", ipAddrClaimName))
			}
		}
	}

	ctrlutil.RemoveFinalizer(ctx.IONOSCloudMachine, v1alpha1.MachineFinalizer)

	return reconcile.Result{}, nil
}

func (r *IONOSCloudMachineReconciler) reconcileDeleteLoadBalancerForwardingRule(ctx *context.MachineContext) error {
	if !clusterutilv1.IsControlPlaneMachine(ctx.Machine) {
		ctx.Logger.Info("Deleting IONOSCloudMachine is not a control plane...no forwarding rule deletion required.")
		return nil
	}

	lbSpec := ctx.IONOSCloudCluster.Spec.LoadBalancer
	nic := ctx.IONOSCloudMachine.NicByLan(lbSpec.TargetLanRef.Name)

	if nic == nil || nic.PrimaryIP == nil {
		return nil
	}

	if rules, _, err := ctx.IONOSClient.GetLoadBalancerForwardingRules(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lbSpec.ID); err != nil {
		return err
	} else {
		targetToDelete := ionoscloud.NetworkLoadBalancerForwardingRuleTarget{
			Ip:     nic.PrimaryIP,
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
						lbSpec.ID,
						*rule.Id,
						properties,
					); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (r *IONOSCloudMachineReconciler) reconcileNormal(ctx *context.MachineContext) (reconcile.Result, error) {
	ctx.Logger.Info("Reconciling IONOSCloudMachine")

	// If the IONOSCloudMachine doesn't have our finalizer, add it.
	ctrlutil.AddFinalizer(ctx.IONOSCloudMachine, v1alpha1.MachineFinalizer)

	if result, err := r.reconcileIPAddressClaims(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.IPAddressClaimCreatedCondition, v1alpha1.IPAddressClaimCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, errors.Wrap(err, "failed reconcileIPAddressClaims")
	} else if result != nil {
		return *result, nil
	}

	if result, err := r.reconcileIPAddresses(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.IPAddressCreatedCondition, v1alpha1.IPAddressCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, errors.Wrap(err, "failed reconcileIPAddresses")
	} else if result != nil {
		return *result, nil
	}

	if result, err := r.reconcileServer(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.ServerCreatedCondition, v1alpha1.ServerCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, errors.Wrap(err, "failed reconcileServer")
	} else if result != nil {
		return *result, nil
	}

	if result, err := r.reconcileLoadBalancerForwardingRule(ctx); err != nil {
		conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.LoadBalancerForwardingRuleCreatedCondition, v1alpha1.LoadBalancerForwardingRuleCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return *result, errors.Wrap(err, "failed reconcileLoadBalancerForwardingRule")
	} else if result != nil {
		return *result, nil
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

// createIPAddressClaim sets up the ipam IPAddressClaim object and creates it in
// the API.
func createIPAddressClaim(ctx *context.MachineContext, ipAddrClaimName string, poolRef *corev1.TypedLocalObjectReference) error {
	ctx.Logger.Info("creating IPAddressClaim", "name", ipAddrClaimName)

	claim := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ipAddrClaimName,
			Namespace: ctx.IONOSCloudMachine.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1", //ctx.IONOSCloudMachine.APIVersion,
					Kind:       "IONOSCloudMachine",                        //ctx.IONOSCloudMachine.Kind,
					Name:       ctx.IONOSCloudMachine.Name,
					UID:        ctx.IONOSCloudMachine.UID,
				},
			},
			Finalizers: []string{v1alpha1.MachineFinalizer},
		},
		Spec: ipamv1.IPAddressClaimSpec{PoolRef: *poolRef},
	}
	return ctx.K8sClient.Create(ctx, claim)
}

// IPAddressClaimName returns a name given a VsphereVM name, deviceIndex, and
// poolIndex.
func IPAddressClaimName(vmName string, deviceIndex, poolIndex int) string {
	return fmt.Sprintf("%s-%d-%d", vmName, deviceIndex, poolIndex)
}

// reconcileIPAddressClaims ensures that IONOSCloudMachines that are configured with
// .spec.nics.PrimaryAddressFrom have corresponding IPAddressClaims.
func (r *IONOSCloudMachineReconciler) reconcileIPAddressClaims(ctx *context.MachineContext) (*reconcile.Result, error) {
	for devIdx, device := range ctx.IONOSCloudMachine.Spec.Nics {
		if device.PrimaryAddressFrom == nil {
			continue
		}
		// check if claim exists
		ipAddrClaim := &ipamv1.IPAddressClaim{}
		ipAddrClaimName := IPAddressClaimName(ctx.IONOSCloudMachine.Name, devIdx, 0)
		ipAddrClaimKey := apitypes.NamespacedName{
			Namespace: ctx.IONOSCloudMachine.Namespace,
			Name:      ipAddrClaimName,
		}
		var err error
		if err = ctx.K8sClient.Get(ctx, ipAddrClaimKey, ipAddrClaim); err != nil && !apierrors.IsNotFound(err) {
			return &reconcile.Result{}, err
		}
		if err == nil {
			ctx.Logger.Info("IPAddressClaim found", "name", ipAddrClaimName)
		}
		if apierrors.IsNotFound(err) {
			if err = createIPAddressClaim(ctx, ipAddrClaimName, device.PrimaryAddressFrom); err != nil {
				ctx.Logger.Error(err, "Creation of ipAddressClaim failed")
				return &reconcile.Result{}, err
			}
		}
	}
	return nil, nil
}

// reconcileIPAddresses prevents successful reconciliation of a IONOSCloudMachine
// until an IPAM Provider updates each IPAddressClaim associated to the
// IONOSCloudMachine with a reference to an IPAddress. This function is a no-op if the
// IONOSCloudMachine has no associated IPAddressClaims.
func (r *IONOSCloudMachineReconciler) reconcileIPAddresses(ctx *context.MachineContext) (*reconcile.Result, error) {
	for devIdx := range ctx.IONOSCloudMachine.Spec.Nics {
		device := &ctx.IONOSCloudMachine.Spec.Nics[devIdx]
		if device.PrimaryAddressFrom == nil {
			continue
		}

		// check if claim exists
		ipAddrClaim := &ipamv1.IPAddressClaim{}
		ipAddrClaimName := IPAddressClaimName(ctx.IONOSCloudMachine.Name, devIdx, 0)
		ipAddrClaimKey := apitypes.NamespacedName{
			Namespace: ctx.IONOSCloudMachine.Namespace,
			Name:      ipAddrClaimName,
		}
		var err error
		ctx.Logger.Info("fetching IPAddressClaim", "name", ipAddrClaimKey.String())
		if err = ctx.K8sClient.Get(ctx, ipAddrClaimKey, ipAddrClaim); err != nil && !apierrors.IsNotFound(err) {
			ctx.Logger.Error(err, "error fetching IPAddressClaim", "name", ipAddrClaimName)
			return &reconcile.Result{}, err
		}

		ipAddrName := ipAddrClaim.Status.AddressRef.Name
		ctx.Logger.Info("fetched IPAddressClaim", "name", ipAddrClaimName, "IPAddressClaim.Status.AddressRef.Name", ipAddrName)
		if ipAddrName == "" {
			ctx.Logger.Info("IPAddress name was empty on IPAddressClaim", "name", ipAddrClaimName, "IPAddressClaim.Status.AddressRef.Name", ipAddrName)
			msg := "Waiting for IPAddressClaim to have an IPAddress bound"
			return &reconcile.Result{}, errors.New(msg)
		}

		ipAddr := &ipamv1.IPAddress{}
		ipAddrKey := apitypes.NamespacedName{
			Namespace: ctx.IONOSCloudMachine.Namespace,
			Name:      ipAddrName,
		}
		if err = ctx.K8sClient.Get(ctx, ipAddrKey, ipAddr); err != nil {
			return &reconcile.Result{}, err
		}

		toAdd := ipAddr.Spec.Address
		ctx.Logger.Info("fetched IPAddress", "ip", toAdd)
		_, err = netip.ParseAddr(toAdd)
		if err != nil {
			msg := fmt.Sprintf("IPAddress %s/%s has invalid ip address: %q",
				ipAddrKey.Namespace,
				ipAddrKey.Name,
				toAdd,
			)
			conditions.MarkFalse(ctx.IONOSCloudMachine, v1alpha1.IPAddressCreatedCondition, msg, clusterv1.ConditionSeverityInfo, "")
			return &reconcile.Result{RequeueAfter: defaultMachineRetryIntervalOnBusy}, err
		}
		device.PrimaryIP = ionoscloud.ToPtr(toAdd)
		conditions.MarkTrue(ctx.IONOSCloudMachine, v1alpha1.IPAddressCreatedCondition)
	}

	return nil, nil
}

func (r *IONOSCloudMachineReconciler) reconcileServer(ctx *context.MachineContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling server")

	if ctx.IONOSCloudMachine.Spec.ProviderID == "" {
		bootstrapData, err := r.getBootstrapData(ctx)
		if err != nil {
			return &reconcile.Result{}, errors.Wrapf(err, "unable to get bootstrap data")
		}
		diskSize, err := utils.ToFloat32(ctx.IONOSCloudMachine.Spec.BootVolume.Size)
		if err != nil {
			return &reconcile.Result{}, errors.Wrapf(err, "invalid spec.bootvolume.disksize")
		}

		nics := make([]ionoscloud.Nic, 0)
		for _, nic := range ctx.IONOSCloudMachine.Spec.Nics {
			lanSpec := ctx.IONOSCloudCluster.Lan(nic.LanRef.Name)
			var ips []string
			if nic.PrimaryAddressFrom != nil {
				if nic.PrimaryIP == nil || *nic.PrimaryIP == "" {
					return &reconcile.Result{}, errors.New("unexpected empty PrimaryIP")
				}
				ips = append(ips, *nic.PrimaryIP)
			}
			nics = append(nics, ionoscloud.Nic{
				Properties: &ionoscloud.NicProperties{
					Dhcp: ionoscloud.ToPtr(true),
					Ips:  &ips,
					Lan:  lanSpec.LanID,
					Name: ionoscloud.ToPtr(fmt.Sprintf("%s-nic-%s", ctx.IONOSCloudMachine.Name, lanSpec.Name)),
				},
			})
		}
		ctx.Logger.Info(fmt.Sprintf("nics: %v", nics))

		server := ionoscloud.Server{
			Entities: &ionoscloud.ServerEntities{
				Nics: &ionoscloud.Nics{
					Items: &nics,
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
		ctx.Logger.Info("server not yet created")
		return &reconcile.Result{RequeueAfter: defaultMachineRetryIntervalOnBusy}, nil
	}
	ipObtained := false
	nics := *server.Entities.Nics.Items
	for _, nic := range nics {
		ips := *nic.Properties.Ips
		lan := ctx.IONOSCloudCluster.LanBy(nic.Properties.Lan)
		if lan == nil || len(ips) == 0 {
			continue
		}
		ipObtained = true
		ctx.IONOSCloudMachine.EnsureNic(v1alpha1.IONOSNicSpec{
			LanRef: v1alpha1.IONOSLanRefSpec{
				Name: lan.Name,
			},
			PrimaryIP: &ips[0],
		})
	}

	if !ipObtained {
		ctx.Logger.Info("server does not have an ip yet")
		return &reconcile.Result{RequeueAfter: defaultMachineRetryIntervalOnBusy}, nil
	}

	conditions.MarkTrue(ctx.IONOSCloudMachine, v1alpha1.ServerCreatedCondition)

	err = r.reconcileFailoverGroups(ctx, server)
	if err != nil {
		ctx.Logger.Error(err, "failoverGroup not yet correct assigned")
		return &reconcile.Result{RequeueAfter: defaultMachineRetryIntervalOnBusy}, nil
	}

	return nil, nil
}

func (r *IONOSCloudMachineReconciler) reconcileFailoverGroups(ctx *context.MachineContext, server ionoscloud.Server) error {
	var multiErr error
	for i := range ctx.IONOSCloudCluster.Spec.Lans {
		lanSpec := &ctx.IONOSCloudCluster.Spec.Lans[i]
		serverNic := serverNicByLan(server, lanSpec)
		if serverNic == nil {
			continue
		}
		for k := range lanSpec.FailoverGroups {
			group := &lanSpec.FailoverGroups[k]
			ctx.Logger.Info("Reconciling failover group " + group.ID)
			block, _, err := ctx.IONOSClient.GetIPBlock(ctx, group.ID)
			if err != nil {
				multiErr = multierr.Append(multiErr, err)
				continue
			}
			ips := *block.Properties.Ips
			err = r.reconcileNicsWithAdditionalIPs(ctx, *serverNic, ctx.IONOSCloudCluster.Spec.DataCenterID, ctx.IONOSCloudMachine.Spec.ProviderID, ips)
			if err != nil {
				multiErr = multierr.Append(multiErr, err)
			}
			lanId := fmt.Sprint(*lanSpec.LanID)
			err = r.reconcileLanWithAdditionalIPFailover(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lanId, ips)
			if err != nil {
				multiErr = multierr.Append(multiErr, err)
			}
		}
	}
	return multiErr
}

func (r *IONOSCloudMachineReconciler) reconcileLoadBalancerForwardingRule(ctx *context.MachineContext) (*reconcile.Result, error) {
	ctx.Logger.Info("Reconciling load balancer forwarding rule")

	if !clusterutilv1.IsControlPlaneMachine(ctx.Machine) {
		ctx.Logger.Info("Reconciled IONOSCloudMachine is not a control plane...no forwarding rule needed.")
		return nil, nil
	}

	lbSpec := ctx.IONOSCloudCluster.Spec.LoadBalancer
	nic := ctx.IONOSCloudMachine.NicByLan(lbSpec.TargetLanRef.Name)

	if nic == nil && nic.PrimaryIP != nil {
		return &reconcile.Result{}, errors.New("server has not obtained an target ip yet")
	}

	if ctx.IONOSCloudMachine.Spec.ProviderID != "" {
		rules, _, err := ctx.IONOSClient.GetLoadBalancerForwardingRules(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lbSpec.ID)
		if err != nil {
			return &reconcile.Result{}, errors.Wrap(err, "error getting forwarding rules")
		}
		desiredTarget := ionoscloud.NetworkLoadBalancerForwardingRuleTarget{
			Ip:     nic.PrimaryIP,
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

				if _, _, err = ctx.IONOSClient.PatchLoadBalancerForwardingRule(ctx, ctx.IONOSCloudCluster.Spec.DataCenterID, lbSpec.ID, *rule.Id, properties); err != nil {
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

func (r *IONOSCloudMachineReconciler) reconcileNicsWithAdditionalIPs(ctx *context.MachineContext, nic ionoscloud.Nic, datacenterId, serverId string, toEnsureIPs []string) error {
	requirePatch := false
	ips := make([]string, 0)
	if nic.Properties.Ips != nil {
		ips = *nic.Properties.Ips
	}
	for _, ip := range toEnsureIPs {
		toAdd := true
		for i := range ips {
			if ips[i] == ip {
				toAdd = false
				break
			}
		}
		if toAdd {
			requirePatch = true
			ips = append(ips, ip)
		}
	}
	if requirePatch {
		return ctx.IONOSClient.PatchServerNicsWithIPs(ctx, datacenterId, serverId, *nic.Id, ips)
	}
	return nil
}

func (r *IONOSCloudMachineReconciler) reconcileLanWithAdditionalIPFailover(ctx *context.MachineContext, datacenterId, lanId string, toEnsureIPs []string) error {
	reqCtx := goctx.WithValue(ctx.Context, ionos.DepthKey, int32(2))
	lan, _, err := ctx.IONOSClient.GetLan(reqCtx, datacenterId, lanId)
	if err != nil {
		return err
	}
	nicUuid := ""
	nics := *lan.Entities.Nics.Items
	for i := range nics {
		nic := &nics[i]
		if nic.Properties.Ips == nil {
			continue
		}
		hasIps := 0
		for k := range toEnsureIPs {
			for _, s := range *nic.Properties.Ips {
				if s == toEnsureIPs[k] {
					hasIps += 1
					break
				}
			}
		}
		if len(toEnsureIPs) == hasIps {
			nicUuid = *nic.Id
			break
		}
	}
	if nicUuid == "" {
		return errors.New("no nic found with all ips")
	}
	ctx.Logger.Info("nic with all ips", "nicUuid", nicUuid, "toEnsureIPs", toEnsureIPs)
	// always override
	requirePatch := false
	failovers := make([]ionoscloud.IPFailover, 0)
	if lan.Properties.IpFailover != nil {
		failovers = append(failovers, *lan.Properties.IpFailover...)
	}
	for k := range toEnsureIPs {
		toAdd := true
		for i := range failovers {
			if *failovers[i].Ip == toEnsureIPs[k] {
				toAdd = false
				break
			}
		}
		if toAdd {
			requirePatch = true
			failovers = append(failovers, ionoscloud.IPFailover{
				Ip:      &toEnsureIPs[k],
				NicUuid: &nicUuid,
			})
		}
	}
	ctx.Logger.Info("patch", "requirePatch", requirePatch, "lanId", lanId, "failovers", failovers)
	if requirePatch {
		return ctx.IONOSClient.PatchLanWithIPFailover(ctx, datacenterId, lanId, failovers)
	}
	return nil
}

func serverNicByLan(server ionoscloud.Server, lan *v1alpha1.IONOSLanSpec) *ionoscloud.Nic {
	var serverNic *ionoscloud.Nic
	for _, nic := range *server.Entities.Nics.Items {
		if *nic.Properties.Lan == *lan.LanID {
			serverNic = &nic
			break
		}
	}
	return serverNic
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
