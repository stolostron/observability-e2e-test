package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	It("should be automatically created within 1 minute when delete manifestwork (manifestwork/g0)", func() {
		manifestWorkName := "endpoint-observability-work"
		clusters, err := dynClient.Resource(utils.NewOCMManagedClustersGVR()).List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		clientDynamic := utils.GetKubeClientDynamic(testOptions, true)
		for _, cluster := range clusters.Items {
			clusterName := cluster.Object["metadata"].(map[string]interface{})["name"].(string)

			By("Waiting for manifestwork to be deleted")
			Eventually(func() error {
				err := clientDynamic.Resource(utils.NewOCMManifestworksGVR()).Namespace(clusterName).Delete(manifestWorkName, &metav1.DeleteOptions{})
				return err
			}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(Succeed())

			By("Waiting for manifestwork to be created automatically")
			Eventually(func() error {
				_, err := clientDynamic.Resource(utils.NewOCMManifestworksGVR()).Namespace(clusterName).Get(manifestWorkName, metav1.GetOptions{})
				return err
			}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())

			By("Checking metric to ensure that no data is lost in 1 minute")
			Eventually(func() error {
				err, _ = utils.ContainManagedClusterMetric(testOptions, "node_memory_MemAvailable_bytes", "1m", []string{`"__name__":"node_memory_MemAvailable_bytes"`})
				return err
			}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*3).Should(Succeed())
		}
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
