package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)


var _ = Describe("Outscale Cluster Reconciler", func() {
	BeforeEach(func() {})
	AfterEach(func() {})
	Context("Reconcile an Outscale cluster", func() {
		It("should create a cluster", func() {
			access_key := os.Getenv("OSC_ACCESS_KEY")
			secret_key := os.Getenv("OSC_SECRET_KEY")
			ctx := context.Background()
			
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "secret",
					Namespace:    "default",
				},
				Data: map[string][]byte{
					access_key: []byte(access_key),
					secret_key: []byte(secret_key),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func(){
				err := k8sClient.Delete(ctx, secret)
				Expect(err).NotTo(HaveOccurred())
			}()
		})
	})
})
