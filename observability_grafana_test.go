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

	It("should be able to access the grafana console (grafana/g0)", func() {
		Eventually(func() error {
			err := utils.CheckGrafanaConsole(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("should have metric data in grafana console (addon/g0) (grafana/g0)", func() {
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions)
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})
})
