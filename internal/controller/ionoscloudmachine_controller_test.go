package controller_test

import (
	"context"
	"fmt"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	fakes "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "github.com/onsi/gomega/gstruct"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("IONOSCloudMachine controller", func() {

	const (
		timeout   = time.Second * 10
		interval  = time.Millisecond * 250
		namespace = "default"
	)

	var (
		fakeClientLookupKey types.UID
		fakeClient          *fakes.FakeClient

		CAPIClusterName string
		CAPIMachineName string
		capiCluster     *clusterv1.Cluster
		capiMachine     *clusterv1.Machine

		CAPICClusterName         string
		CAPICMachineName         string
		CAPICClusterIdentityName string
		IdentitySecretName       string
		BootstrapSecretName      string
		capicCluster             *v1alpha1.IONOSCloudCluster
		capicMachine             *v1alpha1.IONOSCloudMachine
		capicClusterIdentity     *v1alpha1.IONOSCloudClusterIdentity
		identitySecret           *v1.Secret
		bootstrapSecret          *v1.Secret
	)

	BeforeEach(func() {
		fakeClientLookupKey = uuid.NewUUID()
		fakeClient = GetFakeClient(string(fakeClientLookupKey))

		CAPIMachineName = fmt.Sprintf("capi-machine-%s", fakeClientLookupKey)
		CAPICClusterName = fmt.Sprintf("capic-cluster-%s", fakeClientLookupKey)
		CAPICMachineName = fmt.Sprintf("capic-machine-%s", fakeClientLookupKey)
		CAPICClusterIdentityName = fmt.Sprintf("capic-cluster-identity-%s", fakeClientLookupKey)
		CAPIClusterName = fmt.Sprintf("capi-cluster-%s", fakeClientLookupKey)
		IdentitySecretName = fmt.Sprintf("identity-secret-%s", fakeClientLookupKey)
		BootstrapSecretName = fmt.Sprintf("bootstrap-secret-%s", fakeClientLookupKey)
		capiMachine = &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CAPIMachineName,
				Namespace: namespace,
				Labels: map[string]string{
					clusterv1.MachineControlPlaneLabel: "",
				},
				UID: uuid.NewUUID(),
			},
			Spec: clusterv1.MachineSpec{
				ClusterName: CAPIClusterName,
				Bootstrap: clusterv1.Bootstrap{
					DataSecretName: ionoscloud.ToPtr(BootstrapSecretName),
				},
			},
		}
		capicClusterIdentity = &v1alpha1.IONOSCloudClusterIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CAPICClusterIdentityName,
				Namespace: namespace,
			},
			Spec: v1alpha1.IONOSCloudClusterIdentitySpec{
				SecretName: IdentitySecretName,
				HostUrl:    string(fakeClientLookupKey),
			},
		}
		capicCluster = &v1alpha1.IONOSCloudCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "IONOSCloudCluster",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      CAPICClusterName,
				Namespace: namespace,
				UID:       uuid.NewUUID(),
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         "cluster.x-k8s.io/v1beta1",
					Kind:               "Cluster",
					Name:               CAPIClusterName,
					Controller:         ionoscloud.ToPtr(true),
					BlockOwnerDeletion: ionoscloud.ToPtr(true),
					UID:                fakeClientLookupKey,
				}},
			},
			Spec: v1alpha1.IONOSCloudClusterSpec{
				Location: "de/txl",
				ControlPlaneEndpoint: clusterv1.APIEndpoint{
					Host: "123.123.123.123",
					Port: 6443,
				},
				IdentityName: CAPICClusterIdentityName,
			},
		}
		capicMachine = &v1alpha1.IONOSCloudMachine{
			TypeMeta: metav1.TypeMeta{
				Kind:       "IONOSCloudMachine",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.String(),
						Kind:       "Machine",
						Name:       CAPIMachineName,
						UID:        fakeClientLookupKey,
					},
				},
				Name:      CAPICMachineName,
				Namespace: namespace,
			},
			Spec: v1alpha1.IONOSCloudMachineSpec{
				BootVolume: v1alpha1.IONOSVolumeSpec{
					Size: "2048",
				},
			},
		}
		identitySecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      IdentitySecretName,
				Namespace: namespace,
			},
			StringData: map[string]string{"token": string(uuid.NewUUID())},
		}
		bootstrapSecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BootstrapSecretName,
				Namespace: namespace,
			},
			StringData: map[string]string{
				"format": "cloud-config",
				"value":  "xyz",
			},
		}
		capiCluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CAPIClusterName,
				Namespace: namespace,
			},
			Spec: clusterv1.ClusterSpec{
				InfrastructureRef: &v1.ObjectReference{
					Kind:       "IONOSCloudCluster",
					Namespace:  namespace,
					Name:       capicCluster.Name,
					UID:        fakeClientLookupKey,
					APIVersion: v1alpha1.GroupVersion.String(),
				},
			},
		}
	})

	When("IONOSCloudMachine is set up correctly", func() {
		BeforeEach(func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, identitySecret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, bootstrapSecret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capicClusterIdentity)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capiCluster)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capicCluster)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capiMachine)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capicMachine)).Should(Succeed())
		})

		It("should create a Server", func() {
			Eventually(func() int {
				if len(fakeClient.DataCenters) == 0 {
					return 0
				}

				for _, dc := range fakeClient.DataCenters {
					lbs := dc.Entities.Servers.Items
					return len(*lbs)
				}

				return 0
			}, timeout, interval).Should(Equal(1))
		})
	})
})
