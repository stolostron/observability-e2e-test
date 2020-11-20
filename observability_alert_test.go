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

	It("[P1,Sev1,observability]should have custom alert generated (alert/g0)", func() {
		By("Creating custom alert rules")
		cm := utils.CreateCustomAlertRuleYaml("instance:node_memory_utilisation:ratio * 100 > 0")
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, cm)).NotTo(HaveOccurred())

		By("Checking alert generated")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, `ALERTS{alertname="NodeOutOfMemory"}`, "2m", []string{`"__name__":"ALERTS"`, `"alertname":"NodeOutOfMemory"`})
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1,Sev1,observability]should have custom alert updated (alert/g0)", func() {
		By("Updating custom alert rules")
		cm := utils.CreateCustomAlertRuleYaml("instance:node_memory_utilisation:ratio * 100 < 0")
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, cm)).NotTo(HaveOccurred())

		By("Checking alert generated")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, `ALERTS{alertname="NodeOutOfMemory"}`, "1m", []string{`"__name__":"ALERTS"`, `"alertname":"NodeOutOfMemory"`})
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})
})
