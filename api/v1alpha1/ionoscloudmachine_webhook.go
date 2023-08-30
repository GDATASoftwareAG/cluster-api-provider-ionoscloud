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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var ionoscloudmachinelog = logf.Log.WithName("ionoscloudmachine-resource")

func (r *IONOSCloudMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-ionoscloudmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines,verbs=update,versions=v1alpha1,name=vionoscloudmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &IONOSCloudMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *IONOSCloudMachine) ValidateCreate() (admission.Warnings, error) {
	ionoscloudmachinelog.Info("validate create", "name", r.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *IONOSCloudMachine) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	ionoscloudmachinelog.Info("validate update", "name", r.Name)

	//newIONOSCloudMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
	//if err != nil {
	//	return nil, apierrors.NewInternalError(errors.Wrap(err, "failed to convert new IONOSCloudMachine to unstructured object"))
	//}
	//
	//oldIONOSCloudMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(old)
	//if err != nil {
	//	return nil, apierrors.NewInternalError(errors.Wrap(err, "failed to convert old IONOSCloudMachine to unstructured object"))
	//}
	//
	//var allErrs field.ErrorList
	//
	//newVSphereMachineSpec := newIONOSCloudMachine["spec"].(map[string]interface{})
	//oldVSphereMachineSpec := oldIONOSCloudMachine["spec"].(map[string]interface{})
	//
	//// allow changes to providerID
	//delete(oldVSphereMachineSpec, "providerID")
	//delete(newVSphereMachineSpec, "providerID")
	//
	//// allow changes to providerID
	//delete(oldVSphereMachineSpec, "providerID")
	//delete(newVSphereMachineSpec, "providerID")
	//
	//
	//if !reflect.DeepEqual(oldVSphereMachineSpec, newVSphereMachineSpec) {
	//	allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "cannot be modified"))
	//}
	//
	//if allErrs != nil {
	//	return nil, apierrors.NewInvalid(
	//		schema.GroupKind{Group: "infrastructure.cluster.x-k8s.io", Kind: "IONOSCloudMachine"},
	//		r.Name, allErrs)
	//}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *IONOSCloudMachine) ValidateDelete() (admission.Warnings, error) {
	ionoscloudmachinelog.Info("validate delete", "name", r.Name)

	return nil, nil
}
