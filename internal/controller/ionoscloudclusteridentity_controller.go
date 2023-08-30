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
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/utils"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/context"
)

const (
	UserNameProperty = "username"
	PasswordProperty = "password"
	TokenProperty    = "token"
)

// IONOSCloudClusterIdentityReconciler reconciles a IONOSCloudClusterIdentity object
type IONOSCloudClusterIdentityReconciler struct {
	*context.ControllerContext
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusteridentities,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusteridentities/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusteridentities/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *IONOSCloudClusterIdentityReconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)

	identity := &v1alpha1.IONOSCloudClusterIdentity{}
	if err := r.K8sClient.Get(ctx, req.NamespacedName, identity); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Build the patch helper.
	patchHelper, err := patch.NewHelper(identity, r.K8sClient)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to initialize patch helper")
	}

	// Create the cluster context for this request.
	identityContext := &context.IdentityContext{
		ControllerContext:         r.ControllerContext,
		Logger:                    r.Logger.WithName(req.Namespace).WithName(req.Name),
		PatchHelper:               patchHelper,
		IONOSCloudClusterIdentity: identity,
	}

	// Always issue a patch when exiting this function so changes to the
	// resource are patched back to the API server.
	defer func() {
		if err := identityContext.Patch(); err != nil {
			if reterr == nil {
				reterr = err
			}
			identityContext.Logger.Error(err, "patch failed", "cluster", identityContext.String())
		}
	}()

	// Handle deleted identities
	if !identity.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(identityContext)
	}

	// Handle non-deleted identities
	return r.reconcileNormal(identityContext)
}

func (r *IONOSCloudClusterIdentityReconciler) reconcileDelete(ctx *context.IdentityContext) (reconcile.Result, error) {
	ctx.Logger.Info("Deleting IONOSCloudClusterIdentity")

	secret := &v1.Secret{}
	nsn := types.NamespacedName{
		Name:      ctx.IONOSCloudClusterIdentity.Spec.SecretName,
		Namespace: ctx.IONOSCloudClusterIdentity.Namespace,
	}
	err := r.K8sClient.Get(ctx, nsn, secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ctrlutil.RemoveFinalizer(secret, v1alpha1.IdentityFinalizer)
	if err := r.K8sClient.Update(ctx, secret); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.K8sClient.Delete(ctx, secret); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *IONOSCloudClusterIdentityReconciler) reconcileNormal(ctx *context.IdentityContext) (reconcile.Result, error) {
	ctx.Logger.Info("Reconciling IONOSCloudClusterIdentity")

	secret := &v1.Secret{}
	nsn := types.NamespacedName{Name: ctx.IONOSCloudClusterIdentity.Spec.SecretName, Namespace: ctx.IONOSCloudClusterIdentity.Namespace}
	if err := r.K8sClient.Get(ctx, nsn, secret); err != nil {
		if apierrors.IsNotFound(err) {
			conditions.MarkFalse(ctx.IONOSCloudClusterIdentity, v1alpha1.CredentialsAvailableCondition, v1alpha1.SecretNotAvailableReason, clusterv1.ConditionSeverityWarning, err.Error())
			return reconcile.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if err := setOwnerRefOnSecret(ctx, secret); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to set owner ref on secret")
	}

	username := secret.Data[UserNameProperty]
	password := secret.Data[PasswordProperty]
	token := secret.Data[TokenProperty]

	if len(token) > 0 && len(username) == 0 && len(password) == 0 || len(token) == 0 && len(username) > 0 && len(password) > 0 {
		conditions.MarkTrue(ctx.IONOSCloudClusterIdentity, v1alpha1.CredentialsAvailableCondition)
	} else {
		err := errors.New("invalid secret: either provide username/password or token")
		conditions.MarkFalse(ctx.IONOSCloudClusterIdentity, v1alpha1.CredentialsAvailableCondition, v1alpha1.SecretNotAvailableReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{}, err
	}

	cl := r.IONOSClientFactory.GetClient(string(username), string(password), string(token), ctx.IONOSCloudClusterIdentity.Spec.HostUrl)
	_, resp, err := cl.APIInfo(ctx)

	if err != nil {
		conditions.MarkFalse(ctx.IONOSCloudClusterIdentity, v1alpha1.CredentialsValidCondition, v1alpha1.CredentialsInvalidReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{Requeue: utils.Retryable(resp.Response)}, err
	}

	conditions.MarkTrue(ctx.IONOSCloudClusterIdentity, v1alpha1.CredentialsValidCondition)

	ctx.IONOSCloudClusterIdentity.Status.Ready = true

	return ctrl.Result{}, nil
}

func setOwnerRefOnSecret(ctx *context.IdentityContext, secret *v1.Secret) error {
	patchHelper, err := patch.NewHelper(secret, ctx.K8sClient)
	if err != nil {
		return err
	}

	secret.SetOwnerReferences(clusterutilv1.EnsureOwnerRef(
		secret.OwnerReferences,
		metav1.OwnerReference{
			APIVersion: ctx.IONOSCloudClusterIdentity.APIVersion,
			Kind:       ctx.IONOSCloudClusterIdentity.Kind,
			Name:       ctx.IONOSCloudClusterIdentity.Name,
			UID:        ctx.IONOSCloudClusterIdentity.UID,
		}))

	if !ctrlutil.ContainsFinalizer(secret, v1alpha1.IdentityFinalizer) {
		ctrlutil.AddFinalizer(secret, v1alpha1.IdentityFinalizer)
	}

	if err := patchHelper.Patch(ctx, secret); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IONOSCloudClusterIdentityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IONOSCloudClusterIdentity{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
