package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

var _ = Describe("Observability:", func() {
	BeforeEach(func() {
		hubClient = utils.NewKubeClient(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)

		dynClient = utils.NewKubeClientDynamic(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)
	})

	It("[P1,Sev1,observability] should have metrics collector pod restart if cert secret re-generated (certrenew/g0)", func() {
		By("Waiting for metrics collector pod ready")
		podName := ""
		Eventually(func() bool {
			_, podList := utils.GetMetricsCollectorPodList(testOptions)
			if podList != nil && len(podList.Items) > 0 {
				podName = podList.Items[0].Name
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Deleting certificate secret to simulate certificate renew")
		err := utils.DeleteCertSecret(testOptions)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for old metrics collector pod removed: " + podName)
		Eventually(func() bool {
			err, podList := utils.GetMetricsCollectorPodList(testOptions)
			if err == nil {
				for _, pod := range podList.Items {
					if pod.Name == podName {
						return true
					}
				}
			} else {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeFalse())
	})

})
