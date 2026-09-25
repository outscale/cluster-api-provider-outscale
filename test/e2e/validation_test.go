package e2e

import (
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
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
	})

	ginkgo.It("should properly validate OscClusters", func() {
		tts := []struct {
			file  string
			valid bool
		}{
			{file: "testdata/osccluster/invalid/nolbname.yaml"},
		}
		for _, tt := range tts {
			doc, err := load(tt.file)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			err = bootstrapClusterProxy.GetClient().Create(ctx, doc)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	})
})
