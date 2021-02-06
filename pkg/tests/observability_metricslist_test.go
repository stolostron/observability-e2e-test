package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

const (
	whitelistCMname = "observability-metrics-custom-allowlist"
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

	It("[P1][Sev1][Observability] Should have metrics which defined in custom metrics whitelist (metricslist/g0)", func() {
		Skip("Should have metrics which defined in custom metrics whitelist (metricslist/g0)")
		By("Adding custom metrics whitelist configmap")
		yamlB, err := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/metrics/whitelist"})
		Expect(err).ToNot(HaveOccurred())
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		By("Waiting for new added metrics on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes offset 1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1][Sev1][Observability] Should have no metrics after custom metrics whitelist deleted (metricslist/g0)", func() {
		Skip("should Should have no metrics after custom metrics whitelist deleted (metricslist/g0)")
		By("Deleting custom metrics whitelist configmap")
		Eventually(func() error {
			err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(whitelistCMname, &metav1.DeleteOptions{})
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(Succeed())

		By("Waiting for new added metrics disappear on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes offset 1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
