package test

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/controllers"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	//	"os"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	"time"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg              *rest.Config
	k8sClient        client.Client
	testEnv          *envtest.Environment
	reconcileTimeout time.Duration
	clusterToClean   string
)

func init() {
	flag.StringVar(&clusterToClean, "clusterToClean", "", "clustster to clean")
	klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)
	logf.SetLogger(klogr.New())
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

const kubeconfigEnvVar = "KUBECONFIG"

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	utilruntime.Must(infrastructurev1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(controlplanev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme.Scheme))
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	Expect(clusterv1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(infrastructurev1beta1.AddToScheme(scheme.Scheme)).To(Succeed())

	//+kubebuilder:scaffold:scheme

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                  scheme.Scheme,
		LeaderElectionNamespace: "cluster-api-provider-outscale-system",
		LeaderElection:          true,
		LeaderElectionID:        "controller-leader-election-capo",
	})
	Expect(err).ToNot(HaveOccurred())
	err = (&controllers.OscClusterReconciler{
		Client:           k8sManager.GetClient(),
		Recorder:         k8sManager.GetEventRecorderFor("osc-controller"),
		ReconcileTimeout: reconcileTimeout,
	}).SetupWithManager(k8sManager)

	go func() {
		defer GinkgoRecover()
		err := k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
