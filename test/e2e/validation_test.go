package e2e //nolint:testpackage

import (
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type ttc struct {
	file          string
	update        func(doc *unstructured.Unstructured) error
	updateOptions []client.UpdateOption
	valid         bool
}

func doTests(tts []ttc) {
	for _, tt := range tts {
		ginkgo.By("create " + tt.file)
		doc, err := load(tt.file)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		err = bootstrapClusterProxy.GetClient().Create(ctx, doc, client.FieldValidation(metav1.FieldValidationStrict))
		if tt.update != nil {
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("** reload")
			// reloading, it might have been modified
			err = bootstrapClusterProxy.GetClient().Get(ctx, client.ObjectKey{Namespace: doc.GetNamespace(), Name: doc.GetName()}, doc)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("** update")
			err = tt.update(doc)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			err = bootstrapClusterProxy.GetClient().Update(ctx, doc, tt.updateOptions...)
		}
		if tt.valid {
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			gomega.Expect(err).To(gomega.HaveOccurred())
		}
		_ = bootstrapClusterProxy.GetClient().Delete(ctx, doc)
	}
}

const namespace = "cluster-api-test"

var _ = ginkgo.Describe("[e2e][validation][fast] Testing CRD validation", func() {
	ginkgo.BeforeEach(func() {
		gomega.Expect(bootstrapClusterProxy).ToNot(gomega.BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when validating schemas")

		ns := &corev1.Namespace{
			Name: namespace,
		}
		ginkgo.By("Checking namespace")
		gomega.Eventually(func() error {
			if err := bootstrapClusterProxy.GetClient().Create(ctx, ns); err != nil {
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}
			return nil
		}, "5m", "10s").Should(gomega.Succeed(), "Failed to create namespace")

		ginkgo.By("Checking webhook")
		gomega.Eventually(func() bool {
			err := bootstrapClusterProxy.GetClient().Create(ctx, &infrastructurev1beta2.OscClusterTemplate{Name: "test", Namespace: namespace})
			return err == nil
		}, "5m", "10s").Should(gomega.BeTrue(), "Failed to check webhook")
		_ = bootstrapClusterProxy.GetClient().Delete(ctx, &infrastructurev1beta2.OscClusterTemplate{Name: "test", Namespace: namespace})
	})

	ginkgo.It("should properly validate OscClusters", func() {
		tts := []ttc{
			{file: "testdata/osccluster/lb-disabled-invalid.yaml"},
			{file: "testdata/osccluster/lb-name-invalid.yaml"},
			{file: "testdata/osccluster/lb-noname-invalid.yaml"},
			{file: "testdata/osccluster/lb-toolongname-invalid.yaml"},
			{file: "testdata/osccluster/lb-type-invalid.yaml"},

			{file: "testdata/osccluster/net-invalid.yaml"},

			{file: "testdata/osccluster/sg-flow-invalid.yaml"},
			{file: "testdata/osccluster/sg-port-invalid.yaml"},
			{file: "testdata/osccluster/sg-range-invalid.yaml"},

			{file: "testdata/osccluster/subnet-norole-invalid.yaml"},
			{file: "testdata/osccluster/subnet-norolealt-invalid.yaml"},
			{file: "testdata/osccluster/subnet-range-invalid.yaml"},

			{file: "testdata/osccluster/subregion-invalid.yaml"},

			{file: "testdata/osccluster/useexisting-noid-invalid.yaml"},

			{file: "testdata/osccluster/base-valid.yaml", update: func(doc *unstructured.Unstructured) error {
				return unstructured.SetNestedField(doc.Object, "updated", "spec", "loadBalancer", "name")
			}},
			{file: "testdata/osccluster/base-valid.yaml", update: func(doc *unstructured.Unstructured) error {
				return unstructured.SetNestedField(doc.Object, "internal", "spec", "loadBalancer", "type")
			}},

			{file: "testdata/osccluster/minimum-valid.yaml", valid: true},
			{file: "testdata/osccluster/base-valid.yaml", valid: true},
			{file: "testdata/osccluster/useexisting-valid.yaml", valid: true},
			{file: "testdata/osccluster/subnet-nonet-valid.yaml", valid: true},
			{file: "testdata/osccluster/base-valid.yaml", update: func(doc *unstructured.Unstructured) error {
				return unstructured.SetNestedStringSlice(doc.Object, []string{"10.0.0.0/8"}, "spec", "allowFromIPRanges")
			}, valid: true},
		}
		doTests(tts)
	})

	ginkgo.It("should properly validate OscMachineTemplate", func() {
		tts := []ttc{
			{file: "testdata/oscmachinetemplate/vm-subregion-invalid.yaml"},
			{file: "testdata/oscmachinetemplate/vm-type-invalid.yaml"},

			{file: "testdata/oscmachinetemplate/vol-device-invalid.yaml"},
			{file: "testdata/oscmachinetemplate/vol-type-invalid.yaml"},
			{file: "testdata/oscmachinetemplate/vol-multipleroot-invalid.yaml"},
			{file: "testdata/oscmachinetemplate/vol-nodevice-invalid.yaml"},
			{file: "testdata/oscmachinetemplate/vol-nonrootsda1-invalid.yaml"},

			{file: "testdata/oscmachinetemplate/base-valid.yaml", update: func(doc *unstructured.Unstructured) error {
				return unstructured.SetNestedField(doc.Object, "updated", "spec", "template", "spec", "image", "name")
			}},

			{file: "testdata/oscmachinetemplate/minimum-valid.yaml", valid: true},
			{file: "testdata/oscmachinetemplate/base-valid.yaml", valid: true},
			{file: "testdata/oscmachinetemplate/base-valid.yaml", update: func(doc *unstructured.Unstructured) error {
				err := unstructured.SetNestedMap(doc.Object, map[string]any{
					"topology.cluster.x-k8s.io/dry-run": "true",
				}, "metadata", "annotations")
				if err != nil {
					return err
				}
				return unstructured.SetNestedField(doc.Object, "updated", "spec", "template", "spec", "image", "name")
			}, updateOptions: []client.UpdateOption{client.DryRunAll}, valid: true},
		}
		doTests(tts)
	})
})
