package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

const (
	dashboardName        = "sample-dashboard"
	dashboardTitle       = "Sample Dashboard for E2E"
	updateDashboardTitle = "Update Sample Dashboard for E2E"
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

	It("should have custom dashboard which defined in configmap (dashboard/g0)", func() {
		By("Creating custom dashboard configmap")
		yamlB, _ := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/dashboards/sample_custom_dashboard"})
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())
		Eventually(func() bool {
			_, result := utils.ContainDashboard(testOptions, dashboardTitle)
			return result
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("should have update custom dashboard after configmap updated (dashboard/g0)", func() {
		By("Updating custom dashboard configmap")
		yamlB, _ := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/dashboards/update_sample_custom_dashboard"})
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())
		Eventually(func() bool {
			_, result := utils.ContainDashboard(testOptions, dashboardTitle)
			return result
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(BeFalse())
		Eventually(func() bool {
			_, result := utils.ContainDashboard(testOptions, updateDashboardTitle)
			return result
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("should have no custom dashboard in grafana after related configmap removed(dashboard/g0)", func() {
		By("Deleting custom dashboard configmap")
		err = utils.DeleteConfigMap(testOptions, true, dashboardName, MCO_NAMESPACE)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			_, result := utils.ContainDashboard(testOptions, updateDashboardTitle)
			return result
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(BeFalse())
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
