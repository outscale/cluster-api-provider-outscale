package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/util"
)

const (
	KubernetesVersion = "KUBERNETES_VERSION"
)

func Byf(format string, a ...interface{}) {
	By(fmt.Sprintf(format, a...))
}

// setupSpecNamespace create the namespace
func setupSpecNamespace(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string) (*corev1.Namespace, context.CancelFunc) {
	Byf("Creating a namespace for hosting the %q test spec", specName)
	namespace, cancelWatches := framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
		Creator:   clusterProxy.GetClient(),
		ClientSet: clusterProxy.GetClientSet(),
		Name:      fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
		LogFolder: filepath.Join(artifactFolder, "clusters", clusterProxy.GetName()),
	})

	return namespace, cancelWatches
}

// Create the namespace
func createNamespace(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, timeout string, interval string) *corev1.Namespace {
	Byf("Creating a namespace %s for bootstrap cluster", specName)
	namespace := framework.CreateNamespace(ctx, framework.CreateNamespaceInput{
		Creator: clusterProxy.GetClient(),
		Name:    specName,
	}, timeout, interval)
	return namespace
}

// dumpSpecResourcesAndCleanup dump all ressource and delete cluster and namespace
func dumpSpecResourcesAndCleanup(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string, namespace *corev1.Namespace, cancelWatches context.CancelFunc, cluster *clusterv1.Cluster, intervalsGetter func(spec, key string) []interface{}, skipCleanup bool) {
	Byf("Dumping logs from the %q workload cluster", cluster.Name)

	clusterProxy.CollectWorkloadClusterLogs(ctx, cluster.Namespace, cluster.Name, filepath.Join(artifactFolder, "clusters", cluster.Name))

	Byf("Dumping all the Cluster API resources in the %q namespace", namespace.Name)

	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    clusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(artifactFolder, "clusters", clusterProxy.GetName(), "resources"),
	})

	if !skipCleanup {
		Byf("Deleting cluster %s/%s", cluster.Namespace, cluster.Name)
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    clusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, intervalsGetter(specName, "wait-delete-cluster")...)

		Byf("Deleting namespace used for hosting the %q test spec", specName)
		framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
			Deleter: clusterProxy.GetClient(),
			Name:    namespace.Name,
		})
	}
	cancelWatches()
}
