/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	CCMPath      = "CCM"
	CCMResources = "CCM_RESOURCES"

	OutscaleProvider = "OUTSCALE_PROVIDER"
)

// Test suite flags.
var (
	// configPath is the path to the e2e config file.
	configPath string
	// useExistingCluster instructs the test to use the current cluster instead of creating a new one (default discovery rules apply).
	useExistingCluster bool
	// artifactFolder is the folder to store e2e test artifacts.
	artifactFolder string
	// skipCleanup prevents cleanup of test resources e.g. for debug purposes.
	skipCleanup bool

	infraProvider string
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

	// alsoLogToFile enables additional logging to the 'ginkgo-log.txt' file in the artifact folder.
	// These logs also contain timestamps.
	alsoLogToFile bool
)

func init() {
	flag.StringVar(&configPath, "e2e.config", "", "path to the e2e config file")
	flag.StringVar(&artifactFolder, "e2e.artifacts-folder", "", "folder where e2e test artifact should be stored")
	flag.BoolVar(&alsoLogToFile, "e2e.also-log-to-file", false, "if true, ginkgo logs are additionally written to the `ginkgo-log.txt` file in the artifacts folder (including timestamps)")
	flag.BoolVar(&skipCleanup, "e2e.skip-resource-cleanup", false, "if true, the resource cleanup after tests will be skipped")
	flag.BoolVar(&useExistingCluster, "e2e.use-existing-cluster", true, "if true, the test uses the current cluster instead of creating a new one (default discovery rules apply)")
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

	RunSpecs(t, "caposc-e2e")
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
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Before all ParallelNodes.
	Expect(configPath).To(BeAnExistingFile(), "Invalid test suite argument. e2e.config should be an existing file.")
	Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid test suite argument. Can't create e2e.artifacts-folder %q", artifactFolder)

	By("Initializing a runtime.Scheme with all the GVK relevant for this test")
	scheme := initScheme()

	By("Loading the e2e test configuration from " + configPath)
	e2eConfig = loadE2EConfig(ctx, configPath)

	By("Loading the e2e test")
	By("Creating a clusterctl local repository into " + artifactFolder)
	clusterctlConfigPath = createClusterctlLocalRepository(ctx, e2eConfig, filepath.Join(artifactFolder, "repository"))

	By("Setting up the bootstrap cluster")
	bootstrapClusterProvider, bootstrapClusterProxy = setupBootstrapCluster(e2eConfig, scheme, useExistingCluster)

	if !useExistingCluster {
		By("Setting up the cluster api outscale provider credential")
		addCredential("cluster-api-provider-outscale", "cluster-api-provider-outscale-system", "40s", "10s")
	}
	By("Initializing the bootstrap cluster")
	initBootstrapCluster(ctx, bootstrapClusterProxy, e2eConfig, clusterctlConfigPath, artifactFolder)

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
	infraProvider = config.Variables[OutscaleProvider]
	return config
}

// createClusterctlLocalRepository create clusterctl local repository with clusterctlConfig
func createClusterctlLocalRepository(ctx context.Context, config *clusterctl.E2EConfig, repositoryFolder string) string {
	createRepositoryInput := CreateRepositoryInput{
		E2EConfig:        config,
		RepositoryFolder: repositoryFolder,
	}

	Expect(config.Variables).To(HaveKey(capi_e2e.CNIPath), "Missing %s variable in the config", capi_e2e.CNIPath)
	cniPath := config.Variables[capi_e2e.CNIPath]
	Expect(cniPath).To(BeAnExistingFile(), "the %s variable should resolve to an existing file", capi_e2e.CNIPath)
	createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(cniPath, capi_e2e.CNIResources)

	Expect(config.Variables).To(HaveKey("CCM"), "Missing %s variable in the config", CCMPath)
	ccmPath := config.Variables["CCM"]
	Expect(ccmPath).To(BeAnExistingFile(), "the %s variable should resolve to an existing file", CCMPath)
	createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(ccmPath, CCMResources)

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
