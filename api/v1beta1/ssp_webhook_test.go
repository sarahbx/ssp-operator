/*


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

package v1beta1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("SSP Validation", func() {
	var (
		client  client.Client
		objects = make([]runtime.Object, 0)
	)

	JustBeforeEach(func() {
		scheme := runtime.NewScheme()
		// add our own scheme
		SchemeBuilder.AddToScheme(scheme)
		// add more schemes
		v1.AddToScheme(scheme)

		client = fake.NewFakeClientWithScheme(scheme, objects...)
		setClientForWebhook(client)
	})

	Context("creating SSP CR", func() {
		const (
			templatesNamespace = "test-templates-ns"
		)

		BeforeEach(func() {
			objects = append(objects, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:            templatesNamespace,
					ResourceVersion: "1",
				},
			})
		})

		AfterEach(func() {
			objects = make([]runtime.Object, 0)
		})

		Context("when one is already present", func() {
			BeforeEach(func() {
				// add an SSP CR to fake client
				objects = append(objects, &SSP{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-ssp",
						Namespace:       "test-ns",
						ResourceVersion: "1",
					},
					Spec: SSPSpec{
						CommonTemplates: CommonTemplates{
							Namespace: templatesNamespace,
						},
					},
				})
			})

			It("should be rejected", func() {
				ssp := SSP{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ssp2",
						Namespace: "test-ns2",
					},
					Spec: SSPSpec{
						CommonTemplates: CommonTemplates{
							Namespace: templatesNamespace,
						},
					},
				}
				err := ssp.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("creation failed, an SSP CR already exists in namespace test-ns: test-ssp"))
			})
		})

		It("should fail if template namespace does not exist", func() {
			const nonexistingNamespace = "nonexisting-namespace"
			ssp := &SSP{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ssp",
					Namespace: "test-ns",
				},
				Spec: SSPSpec{
					CommonTemplates: CommonTemplates{
						Namespace: nonexistingNamespace,
					},
				},
			}
			err := ssp.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creation failed, the configured namespace for common templates does not exist: " + nonexistingNamespace))
		})
	})

	It("should allow update of commonTemplates.namespace", func() {
		oldSsp := &SSP{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ssp",
				Namespace: "test-ns",
			},
			Spec: SSPSpec{
				CommonTemplates: CommonTemplates{
					Namespace: "old-ns",
				},
			},
		}

		newSsp := oldSsp.DeepCopy()
		newSsp.Spec.CommonTemplates.Namespace = "new-ns"

		err := newSsp.ValidateUpdate(oldSsp)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("DataImportCronTemplates", func() {
		const (
			templatesNamespace = "test-templates-ns"
		)

		BeforeEach(func() {
			objects = append(objects, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:            templatesNamespace,
					ResourceVersion: "1",
				},
			})
		})

		AfterEach(func() {
			objects = make([]runtime.Object, 0)
		})

		table.DescribeTable("validate dataImportCronTemplates", func(namespace, name string, shouldFail bool) {

			oldSSP := &SSP{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ssp",
					Namespace: "test-ns",
				},
				Spec: SSPSpec{
					CommonTemplates: CommonTemplates{
						Namespace: templatesNamespace,
						DataImportCronTemplates: []DataImportCronTemplate{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:      name,
									Namespace: namespace,
								},
							},
						},
					},
				},
			}

			newSSP := oldSSP.DeepCopy()

			By("validating create")
			err := newSSP.ValidateCreate()
			checkExpectedError(err, shouldFail)

			By("validating update")
			err = newSSP.ValidateUpdate(oldSSP)
			checkExpectedError(err, shouldFail)
		},
			table.Entry("no namepsace provided", "", "test-name", false),
			table.Entry("no name provided", GoldenImagesNSname, "", true),
			table.Entry("golden image namespace provided", GoldenImagesNSname, "test-name", false),
			table.Entry("invalid namespace provided", "invalid-namespace", "test-name", true),
		)

	})
})

func checkExpectedError(err error, shouldFail bool) {
	if shouldFail {
		ExpectWithOffset(1, err).To(HaveOccurred())
	} else {
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}
}

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}
