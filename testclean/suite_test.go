/*
Copyright 2022 The Kubernetes Authors.

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

package test

import (
	"context"
	"flag"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework/ginkgoextensions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	k8sClient        client.Client
	testEnv          *envtest.Environment
	reconcileTimeout time.Duration
	clusterToClean   string
	artifactFolder   string
	alsoLogToFile    bool
)

func init() {
	flag.StringVar(&clusterToClean, "clusterToClean", "", "clustster to clean")
	klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)
	logf.SetLogger(klogr.New())
}

func TestAPIs(t *testing.T) {
	g := NewWithT(t)

	RegisterFailHandler(Fail)

	if alsoLogToFile {
		w, err := ginkgoextensions.EnableFileLogging(filepath.Join(artifactFolder, "ginkgo-log.txt"))
		g.Expect(err).ToNot(HaveOccurred())
		defer w.Close()
	}

	RunSpecs(t, "capo testclean")
}

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
	cs, err := services.NewServices()
	Expect(err).ToNot(HaveOccurred())
	err = (&controllers.OscClusterReconciler{
		Client:           k8sManager.GetClient(),
		Cloud:            cs,
		Recorder:         k8sManager.GetEventRecorderFor("osc-controller"),
		ReconcileTimeout: reconcileTimeout,
	}).SetupWithManager(context.Background(), k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err := k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
