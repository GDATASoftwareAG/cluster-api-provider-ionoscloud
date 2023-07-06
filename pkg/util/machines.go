package util

import (
	"context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	apitypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetIONOSCloudMachinesInCluster gets a cluster's IONOSCloudMachine resources.
func GetIONOSCloudMachinesInCluster(
	ctx context.Context,
	controllerClient client.Client,
	namespace, clusterName string) ([]*v1alpha1.IONOSCloudMachine, error) {
	labels := map[string]string{clusterv1.ClusterNameLabel: clusterName}
	machineList := &v1alpha1.IONOSCloudMachineList{}

	if err := controllerClient.List(
		ctx, machineList,
		client.InNamespace(namespace),
		client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	machines := make([]*v1alpha1.IONOSCloudMachine, len(machineList.Items))
	for i := range machineList.Items {
		machines[i] = &machineList.Items[i]
	}

	return machines, nil
}

// GetIONOSCloudMachine gets an IONOSCloudMachine resource for the given CAPI Machine.
func GetIONOSCloudMachine(
	ctx context.Context,
	controllerClient client.Client,
	namespace, machineName string) (*v1alpha1.IONOSCloudMachine, error) {
	machine := &v1alpha1.IONOSCloudMachine{}
	namespacedName := apitypes.NamespacedName{
		Namespace: namespace,
		Name:      machineName,
	}
	if err := controllerClient.Get(ctx, namespacedName, machine); err != nil {
		return nil, err
	}
	return machine, nil
}
