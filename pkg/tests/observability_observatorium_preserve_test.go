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

	Context("[P1][Sev1][Observability] Should revert any manual changes on observatorium cr (observatorium_preserve/g0) -", func() {
		It("Updating observatorium cr", func() {
			crName := "observability-observatorium"
			resourceVersionOld := ""
			replicasOld := int64(3)
			Eventually(func() error {
				cr, err := dynClient.Resource(utils.NewMCOMObservatoriumGVR()).Namespace(MCO_NAMESPACE).Get(crName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				replicasOld = cr.Object["spec"].(map[string]interface{})["rule"].(map[string]interface{})["replicas"].(int64)
				cr.Object["spec"].(map[string]interface{})["rule"].(map[string]interface{})["replicas"] = 1

				resourceVersionOld = cr.Object["metadata"].(map[string]interface{})["resourceVersion"].(string)

				_, err = dynClient.Resource(utils.NewMCOMObservatoriumGVR()).Namespace(MCO_NAMESPACE).Update(cr, metav1.UpdateOptions{})
				return err
			}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(Succeed())

			Eventually(func() bool {
				cr, err := dynClient.Resource(utils.NewMCOMObservatoriumGVR()).Namespace(MCO_NAMESPACE).Get(crName, metav1.GetOptions{})
				if err == nil {
					replicasNew := cr.Object["spec"].(map[string]interface{})["rule"].(map[string]interface{})["replicas"].(int64)
					resourceVersionNew := cr.Object["metadata"].(map[string]interface{})["resourceVersion"].(string)
					if resourceVersionNew != resourceVersionOld &&
						replicasNew == replicasOld {
						return true
					}
				}
				return false
			}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(BeTrue())
		})
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
