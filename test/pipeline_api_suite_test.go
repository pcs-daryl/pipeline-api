package main_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	flows "knative.dev/eventing/pkg/apis/flows/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	duck "knative.dev/pkg/apis/duck/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	namespace = "knative"
)

func TestPipelineApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PipelineApi Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "pkg", "api", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "bin", "k8s",
			fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = serving.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = flows.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = messaging.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = duck.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("creating the namespace")
	ctx := context.Background()
	err = createNamespace(ctx, namespace)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createNamespace(ctx context.Context, namespace string) error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	return k8sClient.Create(ctx, ns)

}
