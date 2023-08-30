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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var ionoscloudclusterlog = logf.Log.WithName("ionoscloudcluster-resource")

func (r *IONOSCloudCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-ionoscloudcluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclusters,verbs=update,versions=v1alpha1,name=vionoscloudcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &IONOSCloudCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *IONOSCloudCluster) ValidateCreate() (admission.Warnings, error) {
	ionoscloudclusterlog.Info("validate create", "name", r.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *IONOSCloudCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	ionoscloudclusterlog.Info("validate update", "name", r.Name)

	oldIONOSCloudCluster, _ := old.(*IONOSCloudCluster)
	var allErrs field.ErrorList
	if oldIONOSCloudCluster.Spec.DataCenterID != "" && r.Spec.DataCenterID != oldIONOSCloudCluster.Spec.DataCenterID {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "dataCenterID"), "cannot be modified"))
	}

	if oldIONOSCloudCluster.Spec.Location != "" && r.Spec.Location != oldIONOSCloudCluster.Spec.Location {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "location"), "cannot be modified"))
	}

	if allErrs != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "infrastructure.cluster.x-k8s.io", Kind: "IONOSCloudCluster"},
			r.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *IONOSCloudCluster) ValidateDelete() (admission.Warnings, error) {
	ionoscloudclusterlog.Info("validate delete", "name", r.Name)

	return nil, nil
}
