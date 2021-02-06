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

	It("[P1][Sev1][Observability] Should have metric data in grafana console (grafana/g0)", func() {
		Skip("Should have metric data in grafana console (grafana/g0)")
		Eventually(func() error {
			err, _ = utils.ContainManagedClusterMetric(testOptions, "node_memory_MemAvailable_bytes", []string{`"__name__":"node_memory_MemAvailable_bytes"`})
			return err
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(Succeed())
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
