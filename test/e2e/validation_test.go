//nolint:testpackage
package e2e

import (
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func load(file string) (*unstructured.Unstructured, error) {
	rawCrd, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("error reading %v: %w", file, err)
	}

	doc := &unstructured.Unstructured{}
	err = kubeyaml.Unmarshal(rawCrd, &doc)
	return doc, err
}

var _ = ginkgo.Describe("[e2e][validation][fast] Testing CRD validation", func() {
	ginkgo.BeforeEach(func() {
		gomega.Expect(bootstrapClusterProxy).ToNot(gomega.BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when validating schemas")

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-api",
			},
		}
		gomega.Eventually(func() error {
			if err := bootstrapClusterProxy.GetClient().Create(ctx, ns); err != nil {
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}
			return nil
		}, "5m", "10s").Should(gomega.Succeed(), "Failed to create namespace")
	})

	ginkgo.It("should properly validate OscClusters", func() {
		tts := []struct {
			file, update string
			valid        bool
		}{
			{file: "testdata/osccluster/invalid/nolbname.yaml"},
		}
		for _, tt := range tts {
			doc, err := load(tt.file)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			err = bootstrapClusterProxy.GetClient().Create(ctx, doc)
			if tt.update != "" {
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				doc, err := load(tt.update)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				err = bootstrapClusterProxy.GetClient().Update(ctx, doc)
			}
			if tt.valid {
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).To(gomega.HaveOccurred())
			}
		}
	})
})
