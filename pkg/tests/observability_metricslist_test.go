package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
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

	It("[P1,Sev1,observability] should have metrics which defined in custom metrics whitelist (metricslist/g0)", func() {
		By("Adding custom metrics whitelist configmap")
		err := utils.CreateMetricsWhitelist(testOptions)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for new added metrics on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes", "1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1,Sev1,observability] should have no metrics after custom metrics whitelist deleted (metricslist/g0)", func() {
		By("Deleting custom metrics whitelist configmap")
		err := utils.DeleteMetricsWhitelist(testOptions)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for new added metrics disappear on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes", "1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})
})
