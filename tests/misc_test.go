package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lifecycleapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"

	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
	validator "kubevirt.io/ssp-operator/internal/operands/template-validator"
)

var _ = Describe("Observed generation", func() {
	BeforeEach(func() {
		strategy.SkipSspUpdateTestsIfNeeded()
	})

	AfterEach(func() {
		strategy.RevertToOriginalSspCr()
		waitUntilDeployed()
	})

	It("[test_id:6058] after deployment observedGeneration equals generation", func() {
		ssp := getSsp()
		Expect(ssp.Status.ObservedGeneration).To(Equal(ssp.Generation))
	})

	It("[test_id:6059] should update observed generation after CR update", func() {
		watch, err := StartWatch(sspListerWatcher)
		Expect(err).ToNot(HaveOccurred())
		defer watch.Stop()

		var newValidatorReplicas int32 = 0
		updateSsp(func(foundSsp *sspv1beta1.SSP) {
			foundSsp.Spec.TemplateValidator.Replicas = &newValidatorReplicas
		})

		// Watch changes until above change
		err = WatchChangesUntil(watch, func(updatedSsp *sspv1beta1.SSP) bool {
			return *updatedSsp.Spec.TemplateValidator.Replicas == newValidatorReplicas &&
				updatedSsp.Generation > updatedSsp.Status.ObservedGeneration
		}, shortTimeout)
		Expect(err).ToNot(HaveOccurred())

		// Watch changes until SSP operator updates ObservedGeneration
		err = WatchChangesUntil(watch, func(updatedSsp *sspv1beta1.SSP) bool {
			return *updatedSsp.Spec.TemplateValidator.Replicas == newValidatorReplicas &&
				updatedSsp.Generation == updatedSsp.Status.ObservedGeneration
		}, shortTimeout)
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:6060] should update observed generation when removing CR", func() {
		watch, err := StartWatch(sspListerWatcher)
		Expect(err).ToNot(HaveOccurred())
		defer watch.Stop()

		ssp := getSsp()
		Expect(apiClient.Delete(ctx, ssp)).ToNot(HaveOccurred())

		// Check for deletion timestamp before the SSP operator notices change
		err = WatchChangesUntil(watch, func(updatedSsp *sspv1beta1.SSP) bool {
			return updatedSsp.DeletionTimestamp != nil &&
				updatedSsp.Generation > updatedSsp.Status.ObservedGeneration
		}, shortTimeout)
		Expect(err).ToNot(HaveOccurred())

		// SSP operator enters Deleting phase
		err = WatchChangesUntil(watch, func(updatedSsp *sspv1beta1.SSP) bool {
			return updatedSsp.DeletionTimestamp != nil &&
				updatedSsp.Status.Phase == lifecycleapi.PhaseDeleting &&
				updatedSsp.Generation == updatedSsp.Status.ObservedGeneration
		}, shortTimeout)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("SCC annotation", func() {
	const (
		sccAnnotation = "openshift.io/scc"
		sccRestricted = "restricted"
	)

	BeforeEach(func() {
		waitUntilDeployed()
	})

	It("[test_id:7162] operator pod should have 'restricted' scc annotation", func() {
		pods := &core.PodList{}
		err := apiClient.List(ctx, pods, client.MatchingLabels{"control-plane": "ssp-operator"})

		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())

		for _, pod := range pods.Items {
			Expect(pod.Annotations).To(HaveKeyWithValue(sccAnnotation, sccRestricted), "Expected pod %s/%s to have scc 'restricted'", pod.Namespace, pod.Name)
		}
	})

	It("[test_id:7163] template validator pods should have 'restricted' scc annotation", func() {
		pods := &core.PodList{}
		err := apiClient.List(ctx, pods,
			client.InNamespace(strategy.GetNamespace()),
			client.MatchingLabels{validator.KubevirtIo: validator.VirtTemplateValidator})

		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())

		for _, pod := range pods.Items {
			Expect(pod.Annotations).To(HaveKeyWithValue(sccAnnotation, sccRestricted), "Expected pod %s/%s to have scc 'restricted'", pod.Namespace, pod.Name)
		}
	})
})
