package controller_test

import (
	"context"
	"fmt"
	"time"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	fakes "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/testing"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("IONOSCloudCluster controller", func() {

	const (
		timeout   = time.Second * 10
		interval  = time.Millisecond * 250
		namespace = "default"
	)

	var (
		fakeClientLookupKey string
		fakeClient          *fakes.FakeClient

		CAPIClusterName string
		capiCluster     *clusterv1.Cluster

		CAPICClusterName         string
		CAPICClusterIdentityName string
		IdentitySecretName       string
		capicCluster             *v1alpha1.IONOSCloudCluster
		capicClusterIdentity     *v1alpha1.IONOSCloudClusterIdentity
		identitySecret           *v1.Secret
	)

	BeforeEach(func() {
		fakeClientLookupKey = string(uuid.NewUUID())
		fakeClient = GetFakeClient(fakeClientLookupKey)

		// capi
		CAPIClusterName = fmt.Sprintf("capi-cluster-%s", fakeClientLookupKey)
		capiCluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CAPIClusterName,
				Namespace: namespace,
			},
			Spec: clusterv1.ClusterSpec{},
		}

		// capic
		CAPICClusterName = fmt.Sprintf("capic-cluster-%s", fakeClientLookupKey)
		CAPICClusterIdentityName = fmt.Sprintf("capic-cluster-identity-%s", fakeClientLookupKey)
		IdentitySecretName = fmt.Sprintf("ionos-credentials--%s", fakeClientLookupKey)
		capicClusterIdentity = &v1alpha1.IONOSCloudClusterIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CAPICClusterIdentityName,
				Namespace: namespace,
			},
			Spec: v1alpha1.IONOSCloudClusterIdentitySpec{
				SecretName: IdentitySecretName,
				HostUrl:    fakeClientLookupKey,
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
		identitySecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      IdentitySecretName,
				Namespace: secretNamespace,
			},
			StringData: map[string]string{"token": string(uuid.NewUUID())},
		}
	})

	When("IONOSCloudCluster has Owner and is not paused", func() {
		BeforeEach(func() {
			var uuid = uuid.NewUUID()
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, identitySecret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capicClusterIdentity)).Should(Succeed())
			capiCluster.SetUID(uuid)
			Expect(k8sClient.Create(ctx, capiCluster)).Should(Succeed())
			capicCluster.OwnerReferences = []metav1.OwnerReference{{
				APIVersion:         "cluster.x-k8s.io/v1beta1",
				Kind:               "Cluster",
				Name:               CAPIClusterName,
				Controller:         ionoscloud.ToPtr(true),
				BlockOwnerDeletion: ionoscloud.ToPtr(true),
				UID:                uuid,
			}}
			Expect(k8sClient.Create(ctx, capicCluster)).Should(Succeed())
		})

		It("should set OwnerRefs on corresponding IONOSCloudMachines", func() {
			const (
				machineName = "owned-machine"
			)
			ctx := context.Background()
			machine := &v1alpha1.IONOSCloudMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      machineName,
					Namespace: namespace,
					Labels: map[string]string{
						clusterv1.ClusterNameLabel: CAPIClusterName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())
			lookupKey := types.NamespacedName{Name: machineName, Namespace: namespace}

			Eventually(func() []metav1.OwnerReference {
				createdMachine := &v1alpha1.IONOSCloudMachine{}
				err := k8sClient.Get(ctx, lookupKey, createdMachine)
				if err != nil {
					return nil
				}
				return createdMachine.OwnerReferences
			}, timeout, interval).
				Should(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Kind": Equal("IONOSCloudCluster"),
					"Name": Equal(CAPICClusterName),
				})))
		})

		It("should create Datacenter", func() {
			Eventually(func() int {
				return len(fakeClient.DataCenters)
			}, timeout, interval).Should(Equal(1))
		})

		It("should create LoadBalancer", func() {
			Eventually(func() int {
				if len(fakeClient.DataCenters) == 0 {
					return 0
				}

				for _, dc := range fakeClient.DataCenters {
					lbs := dc.Entities.Networkloadbalancers.Items
					return len(*lbs)
				}

				return 0
			}, timeout, interval).Should(Equal(1))
		})
	})
})
