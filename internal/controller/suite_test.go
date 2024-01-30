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

package controller_test

import (
	goctx "context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/ionos"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/utils"
	fakes "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/testing"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/controller"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrastructurev1alpha1 "github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"

	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/internal/context"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg             *rest.Config
	k8sClient       client.Client
	testEnv         *envtest.Environment
	ctx             goctx.Context
	cancel          goctx.CancelFunc
	FakeClients     = map[string]*fakes.FakeClient{}
	secretNamespace string
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	secretNamespace = utils.IdentitySecretNamespace()
	ctx, cancel = goctx.WithCancel(goctx.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "config", "crd", "external"),
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	requiredSchemes := []func(s *runtime.Scheme) error{
		v1beta1.AddToScheme,
		infrastructurev1alpha1.AddToScheme,
		ipamv1.AddToScheme,
	}

	for _, requiredScheme := range requiredSchemes {
		err = requiredScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
	}
	err = infrastructurev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&controller.IONOSCloudMachineReconciler{
		ControllerContext: &context.ControllerContext{
			Context:            goctx.Background(),
			K8sClient:          k8sManager.GetClient(),
			Scheme:             k8sManager.GetScheme(),
			IONOSClientFactory: ionos.NewClientFactory(NewFakeAPIClient),
		}}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controller.IONOSCloudClusterReconciler{
		ControllerContext: &context.ControllerContext{
			Context:            goctx.Background(),
			K8sClient:          k8sManager.GetClient(),
			Scheme:             k8sManager.GetScheme(),
			IONOSClientFactory: ionos.NewClientFactory(NewFakeAPIClient),
		}}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controller.IONOSCloudClusterIdentityReconciler{
		ControllerContext: &context.ControllerContext{
			Context:            goctx.Background(),
			K8sClient:          k8sManager.GetClient(),
			Scheme:             k8sManager.GetScheme(),
			IONOSClientFactory: ionos.NewClientFactory(NewFakeAPIClient),
		}}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// create namespace
	ctx := goctx.Background()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: secretNamespace},
	}
	Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

func GetFakeClient(key string) *fakes.FakeClient {
	if _, ok := FakeClients[key]; !ok {
		FakeClients[key] = fakes.NewFakeClient()
	}
	return FakeClients[key]
}

func NewFakeAPIClient(_, _, _, host string) ionos.Client {
	return GetFakeClient(host)
}

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := (func() (err error) {
		// Need to sleep if the first stop fails due to a bug:
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1571
		sleepTime := 1 * time.Millisecond
		for i := 0; i < 12; i++ { // Exponentially sleep up to ~4s
			if err = testEnv.Stop(); err == nil {
				return
			}
			sleepTime *= 2
			time.Sleep(sleepTime)
		}
		return
	})()
	Expect(err).NotTo(HaveOccurred())
})
