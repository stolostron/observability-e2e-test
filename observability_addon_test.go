package main_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	It("should have not the expected MCO addon pods (addon/g0)", func() {
		By("Modifying MCO cr to disable observabilityaddon")
		err := utils.ModifyMCOAddonSpec(testOptions, false)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for MCO addon components scales to 0")
		Eventually(func() error {
			addonLabel := "component=metrics-collector"
			var podList, _ = hubClient.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{LabelSelector: addonLabel})
			if len(podList.Items) != 0 {
				return fmt.Errorf("Failed to disable observability addon")
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("should have not metric data (addon/g0)", func() {
		By("Waiting for check no metric data in grafana console")
		Eventually(func() error {
			err, hasMetric := utils.ContainManagedClusterMetric(testOptions, "90s")
			if err != nil && !hasMetric && strings.Contains(err.Error(), "Failed to find metric name from response") {
				return nil
			}
			return fmt.Errorf("Check no metric data in grafana console error: %v", err)
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())

		By("Modifying MCO cr to enalbe observabilityaddon")
		err := utils.ModifyMCOAddonSpec(testOptions, true)
		Expect(err).ToNot(HaveOccurred())
	})
})
