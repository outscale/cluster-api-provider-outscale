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

package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	utils "github.com/outscale/cluster-api-provider-outscale/test/e2e/utils"
	"k8s.io/apimachinery/pkg/runtime"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/test/framework/ginkgoextensions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const kubeconfigEnvVar = "KUBECONFIG"

// Test suite flags.
var (
	// configPath is the path to the e2e config file.
	configPath string

	// useExistingCluster instructs the test to use the current cluster instead of creating a new one (default discovery rules apply).
	useExistingCluster bool

	useCni bool

	useCcm        bool
	validateStack bool
	// artifactFolder is the folder to store e2e test artifacts.
	artifactFolder string

	// skipCleanup prevents cleanup of test resources e.g. for debug purposes.
	skipCleanup bool
)

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	cancel    context.CancelFunc
)

// Test suite global vars.
var (
	ctx = ctrl.SetupSignalHandler()
	// e2eConfig to be used for this test, read from configPath.
	e2eConfig *clusterctl.E2EConfig

	// clusterctlConfigPath to be used for this test, created by generating a clusterctl local repository
	// with the providers specified in the configPath.
	clusterctlConfigPath string

	// bootstrapClusterProvider manages provisioning of the the bootstrap cluster to be used for the e2e tests.
	// Please note that provisioning will be skipped if e2e.use-existing-cluster is provided.
	bootstrapClusterProvider bootstrap.ClusterProvider

	// bootstrapClusterProxy allows to interact with the bootstrap cluster to be used for the e2e tests.
	bootstrapClusterProxy framework.ClusterProxy

	// kubetestConfigFilePath is the path to the kubetest configuration file
	kubetestConfigFilePath string

	// useCIArtifacts specifies whether or not to use the latest build from the main branch of the Kubernetes repository
	useCIArtifacts bool

	// alsoLogToFile enables additional logging to the 'ginkgo-log.txt' file in the artifact folder.
	// These logs also contain timestamps.
	alsoLogToFile bool
)

func init() {
	flag.StringVar(&configPath, "e2e.config", "", "path to the e2e config file")
	flag.StringVar(&artifactFolder, "e2e.artifacts-folder", "", "folder where e2e test artifact should be stored")
	flag.BoolVar(&skipCleanup, "e2e.skip-resource-cleanup", false, "if true, the resource cleanup after tests will be skipped")
	flag.BoolVar(&useExistingCluster, "e2e.use-existing-cluster", true, "if true, the test uses the current cluster instead of creating a new one (default discovery rules apply)")
	flag.BoolVar(&useCni, "e2e.use-cni", false, "if true, the test will use cni clusterclass")
	flag.BoolVar(&useCcm, "e2e.use-ccm", false, "if true, the test will use ccm clusterclass")
	flag.BoolVar(&validateStack, "e2e.validate-stack", false, "if true, the test will validate stack")
	flag.BoolVar(&useCIArtifacts, "kubetest.use-ci-artifacts", false, "use the latest build from the main branch of the Kubernetes repository. Set KUBERNETES_VERSION environment variable to latest-1.xx to use the build from 1.xx release branch.")

	flag.StringVar(&kubetestConfigFilePath, "kubetest.config-file", "", "path to the kubetest configuration file")
}

func TestE2E(t *testing.T) {
	g := NewWithT(t)

	RegisterFailHandler(Fail)

	if alsoLogToFile {
		w, err := ginkgoextensions.EnableFileLogging(filepath.Join(artifactFolder, "ginkgo-log.txt"))
		g.Expect(err).ToNot(HaveOccurred())
		defer w.Close()
	}

	RunSpecs(t, "capo-e2e")
}

func getK8sClient() {
	if os.Getenv(kubeconfigEnvVar) == "" {
		kubeconfig := filepath.Join("/root", ".kube", "config")
		os.Setenv(kubeconfigEnvVar, kubeconfig)
	}
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	testEnv = &envtest.Environment{}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	retryPeriod := 30 * time.Second
	leaseDuration := 80 * time.Second
	renewDeadline := 20 * time.Second
	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaseDuration: &leaseDuration,
		RenewDeadline: &renewDeadline,
		RetryPeriod:   &retryPeriod,
	})
	Expect(err).ToNot(HaveOccurred())
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()
	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())
}

func addCredential(name string, namespace string, timeout string, interval string) {
	const oscAccessKeyEnvVar = "OSC_ACCESS_KEY"
	const oscSecretKeyEnvVar = "OSC_SECRET_KEY"
	const oscRegionKeyEnvVar = "OSC_REGION"
	accessKey := os.Getenv(oscAccessKeyEnvVar)
	secretKey := os.Getenv(oscSecretKeyEnvVar)
	oscRegionKey := os.Getenv(oscRegionKeyEnvVar)
	if bootstrapClusterProxy != nil {
		_ = createNamespace(ctx, namespace, bootstrapClusterProxy, timeout, interval)
		k8sClient := bootstrapClusterProxy.GetClient()
		utils.WaitForCreateMultiSecretAvailable(ctx, utils.CreateMultiSecretInput{
			Getter:          k8sClient,
			Name:            name,
			Namespace:       namespace,
			DataFirstKey:    "access_key",
			DataFirstValue:  accessKey,
			DataSecondKey:   "secret_key",
			DataSecondValue: secretKey,
			DataThirdKey:    "region",
			DataThirdValue:  oscRegionKey,
		})
	}
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// Before all ParallelNodes.
	Expect(configPath).To(BeAnExistingFile(), "Invalid test suite argument. e2e.config should be an existing file.")
	Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid test suite argument. Can't create e2e.artifacts-folder %q", artifactFolder)

	By("Initializing a runtime.Scheme with all the GVK relevant for this test")
	scheme := initScheme()

	By(fmt.Sprintf("Loading the e2e test configuration from %q", configPath))
	e2eConfig = loadE2EConfig(ctx, configPath)

	By("Loading the e2e test")
	By(fmt.Sprintf("Creating a clusterctl local repositorry into %q", artifactFolder))
	clusterctlConfigPath = createClusterctlLocalRepository(ctx, e2eConfig, filepath.Join(artifactFolder, "repository"), useCni)

	By("Setting up the bootstrap cluster")
	bootstrapClusterProvider, bootstrapClusterProxy = setupBootstrapCluster(e2eConfig, scheme, useExistingCluster)

	if validateStack {
		getK8sClient()
	}

	if !useExistingCluster {
		By("Setting up the cluster api outscale provider credential")
		addCredential("cluster-api-provider-outscale", "cluster-api-provider-outscale-system", "40s", "10s")
	}
	By("Initializing the bootstrap cluster")
	initBootstrapCluster(ctx, bootstrapClusterProxy, e2eConfig, clusterctlConfigPath, artifactFolder)

	if os.Getenv(kubeconfigEnvVar) == "" {
		kubeconfig := filepath.Join("/root", ".kube", "config")
		os.Setenv(kubeconfigEnvVar, kubeconfig)
	}
	return []byte(
		strings.Join([]string{
			artifactFolder,
			configPath,
			clusterctlConfigPath,
			bootstrapClusterProxy.GetKubeconfigPath(),
		}, ","),
	)
}, func(data []byte) {
	// Before each ParallelNode.
	parts := strings.Split(string(data), ",")
	Expect(parts).To(HaveLen(4))

	artifactFolder = parts[0]
	configPath = parts[1]
	clusterctlConfigPath = parts[2]
	kubeconfigPath := parts[3]

	e2eConfig = loadE2EConfig(ctx, configPath)
	bootstrapClusterProxy = framework.NewClusterProxy("bootstrap", kubeconfigPath, initScheme())
})

var _ = SynchronizedAfterSuite(func() {
	// After each ParallelNode.
}, func() {
	// After all ParallelNodes.

	By("Tearing down the management cluster")
	if !skipCleanup {
		tearDown(ctx, bootstrapClusterProvider, bootstrapClusterProxy)
	}
	if validateStack {
		cancel()
		By("Tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
})

func initScheme() *runtime.Scheme {
	sc := runtime.NewScheme()
	framework.TryAddDefaultSchemes(sc)
	Expect(infrastructurev1beta1.AddToScheme(sc)).To(Succeed())

	return sc
}

// loadE2EConfig load e2e config
func loadE2EConfig(ctx context.Context, configPath string) *clusterctl.E2EConfig {
	config := clusterctl.LoadE2EConfig(ctx, clusterctl.LoadE2EConfigInput{ConfigPath: configPath})
	Expect(config).ToNot(BeNil(), "Failed to load E2E config from %s", configPath)

	return config
}

// createClusterctlLocalRepository create clusterctl local repository with clusterctlConfig
func createClusterctlLocalRepository(ctx context.Context, config *clusterctl.E2EConfig, repositoryFolder string, useCni bool) string {
	createRepositoryInput := CreateRepositoryInput{
		E2EConfig:        config,
		RepositoryFolder: repositoryFolder,
	}

	if useCni {
		By("Find CNI")
		Expect(config.Variables).To(HaveKey(capi_e2e.CNIPath), "Missing %s variable in the config", capi_e2e.CNIPath)
		cniPath := config.GetVariable(capi_e2e.CNIPath)
		Expect(cniPath).To(BeAnExistingFile(), "the %s variable should resolve to an existing file", capi_e2e.CNIPath)
		By(fmt.Sprintf("Find path %s", cniPath))
		createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(cniPath, "CNI_RESOURCES")
	}

	if useCcm {
		By("Find CCm")
		Expect(config.Variables).To(HaveKey("CCM"), "Missing %s variable in the config", "CCM")
		ccmPath := config.GetVariable("CCM")
		Expect(ccmPath).To(BeAnExistingFile(), "the %s variable should resolve to an existing file", "CCM")
		By(fmt.Sprintf("Find path %s", ccmPath))
		createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(ccmPath, "CCM_RESOURCES")
	}
	clusterctlConfig := CreateRepository(ctx, createRepositoryInput)
	Expect(clusterctlConfig).To(BeAnExistingFile(), "the clusterctl config file does not exists in the local repository %s", repositoryFolder)

	return clusterctlConfig
}

// setupBootstrapCluster will configure bootstrapCluster (i.e clusterctl config)
func setupBootstrapCluster(config *clusterctl.E2EConfig, scheme *runtime.Scheme, useExistingCluster bool) (bootstrap.ClusterProvider, framework.ClusterProxy) {
	var clusterProvider bootstrap.ClusterProvider
	kubeconfigPath := ""
	if !useExistingCluster {
		clusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(context.TODO(), bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
			Name:               config.ManagementClusterName,
			RequiresDockerSock: config.HasDockerProvider(),
			Images:             config.Images,
		})
		Expect(clusterProvider).ToNot(BeNil(), "Failed to create a bootstrap cluster")

		kubeconfigPath = clusterProvider.GetKubeconfigPath()
		Expect(kubeconfigPath).To(BeAnExistingFile(), "Failed to get the kubeconfig file for the bootstrap cluster")
	}

	clusterProxy := framework.NewClusterProxy("bootstrap", kubeconfigPath, scheme)
	Expect(clusterProxy).ToNot(BeNil(), "Failed to get a bootstrap cluster proxy")

	return clusterProvider, clusterProxy
}

// initBootstrapCluster will initialize boostrapcluster (i.e clusterctl init)
func initBootstrapCluster(ctx context.Context, bootstrapClusterProxy framework.ClusterProxy, config *clusterctl.E2EConfig, clusterctlConfig, artifactFolder string) {
	clusterctl.InitManagementClusterAndWatchControllerLogs(ctx, clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		ClusterProxy:            bootstrapClusterProxy,
		ClusterctlConfigPath:    clusterctlConfig,
		InfrastructureProviders: config.InfrastructureProviders(),
		LogFolder:               filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
	}, config.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
}

// tearDown teardown boostrapCluster
func tearDown(ctx context.Context, bootstrapClusterProvider bootstrap.ClusterProvider, bootstrapClusterProxy framework.ClusterProxy) {
	if bootstrapClusterProxy != nil {
		bootstrapClusterProxy.Dispose(ctx)
	}
	if bootstrapClusterProvider != nil {
		bootstrapClusterProvider.Dispose(ctx)
	}
}
