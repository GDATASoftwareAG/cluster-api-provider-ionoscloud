package controller_test

import (
	"context"
	"fmt"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	fakes "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"time"
)

var _ = Describe("IONOSCloudClusterIdentity controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		fakeClientLookupKey string
		fakeClient          *fakes.FakeClient

		CAPICClusterIdentityName string
		IdentitySecretName       string
		capicClusterIdentity     *v1alpha1.IONOSCloudClusterIdentity
		identitySecret           *v1.Secret

		queryClusterIdentityConditions = func() []clusterv1.Condition {
			identity := &v1alpha1.IONOSCloudClusterIdentity{}
			nsn := types.NamespacedName{
				Name: CAPICClusterIdentityName,
			}
			err := k8sClient.Get(ctx, nsn, identity)
			if err != nil {
				return nil
			}
			return identity.Status.Conditions
		}
	)

	BeforeEach(func() {
		fakeClientLookupKey = string(uuid.NewUUID())
		fakeClient = GetFakeClient(fakeClientLookupKey)

		// capic
		CAPICClusterIdentityName = fmt.Sprintf("capic-cluster-identity-%s", fakeClientLookupKey)
		IdentitySecretName = fmt.Sprintf("ionos-credentials--%s", fakeClientLookupKey)
		capicClusterIdentity = &v1alpha1.IONOSCloudClusterIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name: CAPICClusterIdentityName,
			},
			Spec: v1alpha1.IONOSCloudClusterIdentitySpec{
				SecretName: IdentitySecretName,
				HostUrl:    fakeClientLookupKey,
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

	When("IONOSCloudClusterIdentity has a valid secret", func() {
		BeforeEach(func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, identitySecret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, capicClusterIdentity)).Should(Succeed())
		})

		It("should have CredentialsAvailableCondition condition", func() {
			Eventually(queryClusterIdentityConditions, timeout, interval).
				Should(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(v1alpha1.CredentialsAvailableCondition),
					"Status": Equal(v1.ConditionTrue),
				})))
		})

		When("credentials are valid", func() {
			BeforeEach(func() {
				fakeClient.CredentialsAreValid = true
			})

			It("should have CredentialsValidCondition set to true", func() {
				Eventually(queryClusterIdentityConditions, timeout, interval).
					Should(ContainElement(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(v1alpha1.CredentialsValidCondition),
						"Status": Equal(v1.ConditionTrue),
					})))
			})

			It("should have status.ready set to true", func() {
				Eventually(func() bool {
					identity := &v1alpha1.IONOSCloudClusterIdentity{}
					nsn := types.NamespacedName{
						Name: CAPICClusterIdentityName,
					}
					err := k8sClient.Get(ctx, nsn, identity)
					if err != nil {
						return false
					}
					return identity.Status.Ready
				}, timeout, interval).
					Should(BeTrue())
			})
		})

		When("credentials are invalid", func() {
			BeforeEach(func() {
				fakeClient.CredentialsAreValid = false
			})

			It("should have CredentialsValidCondition set to false", func() {
				Eventually(queryClusterIdentityConditions, timeout, interval).
					Should(ContainElement(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(v1alpha1.CredentialsValidCondition),
						"Status": Equal(v1.ConditionFalse),
					})))
			})
		})
	})
})
