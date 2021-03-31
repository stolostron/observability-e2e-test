package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

const (
	MCO_OPERATOR_NAMESPACE = "open-cluster-management"
	MCO_CR_NAME            = "observability"
	MCO_NAMESPACE          = "open-cluster-management-observability"
	MCO_ADDON_NAMESPACE    = "open-cluster-management-addon-observability"
	MCO_LABEL              = "name=multicluster-observability-operator"
	MCO_LABEL_OWNER        = "owner=multicluster-observability-operator"
)

var (
	EventuallyTimeoutMinute  time.Duration = 60 * time.Second
	EventuallyIntervalSecond time.Duration = 1 * time.Second

	hubClient kubernetes.Interface
	dynClient dynamic.Interface
	err       error
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

	It("[P1][Sev1][Observability] Modifying MCO CR for reconciling (reconcile/g0)", func() {
		By("Modifying MCO CR for reconciling")
		err := utils.ModifyMCOCR(testOptions)
		Expect(err).ToNot(HaveOccurred())
	})

	It("[P1][Sev1][Observability] Modifying retentionResolutionRaw (reconcile/g0)", func() {
		By("Waiting for MCO retentionResolutionRaw filed to take effect")
		Eventually(func() error {
			name := MCO_CR_NAME + "-observatorium-thanos-compact"
			compact, getError := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			if getError != nil {
				return getError
			}
			argList := compact.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--retention.resolution-raw=3d" {
					return nil
				}
			}
			return fmt.Errorf("Failed to find modified retention field")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1][Sev1][Observability] Checking node selector for all pods (reconcile/g0)", func() {
		By("Checking node selector for all pods")
		Eventually(func() error {
			err = utils.CheckAllPodNodeSelector(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1][Sev1][Observability] Checking podAntiAffinity for all pods (reconcile/g0)", func() {
		By("Checking podAntiAffinity for all pods")
		Eventually(func() error {
			err := utils.CheckAllPodsAffinity(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1][Sev1][Observability] Revert MCO CR changes (reconcile/g0)", func() {
		if !utils.IsCanaryEnvironment(testOptions) {
			Skip("should skip the high basic mode (reconcile/g0)")
		}
		By("Revert MCO CR changes")
		err := utils.RevertMCOCRModification(testOptions)
		Expect(err).ToNot(HaveOccurred())

		By("Checking MCO components in High mode")
		Eventually(func() error {
			err = utils.CheckMCOComponentsInHighMode(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	AfterEach(func() {
		if testFailed {
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
