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

func testMCOReconcile() {

	hubClient = utils.NewKubeClient(
		testOptions.HubCluster.MasterURL,
		testOptions.KubeConfig,
		testOptions.HubCluster.KubeContext)

	dynClient = utils.NewKubeClientDynamic(
		testOptions.HubCluster.MasterURL,
		testOptions.KubeConfig,
		testOptions.HubCluster.KubeContext)

	By("Modifying MCO retentionResolutionRaw filed")
	err := utils.ModifyMCORetentionResolutionRaw(testOptions)
	Expect(err).ToNot(HaveOccurred())

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

	By("Adding node selector to MCO cr")
	selector := map[string]string{"kubernetes.io/os": "linux"}
	err := utils.ModifyMCONodeSelector(testOptions, selector)
	Expect(err).ToNot(HaveOccurred())

	By("Checking node selector for all pods")
	Eventually(func() error {
		err = utils.CheckAllPodNodeSelector(testOptions)
		if err != nil {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Deleting node selector from MCO cr")
	err = utils.ModifyMCONodeSelector(testOptions, map[string]string{})
	Expect(err).ToNot(HaveOccurred())

	By("Checking podAntiAffinity for all pods")
	Eventually(func() error {
		err := utils.CheckAllPodsAffinity(testOptions)
		if err != nil {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	if !utils.IsCanaryEnvironment(testOptions) {
		Skip("should work in basic mode (reconcile/g0)")
	} else {
		By("Modifying MCO availabilityConfig to enable basic mode")
		Eventually(func() error {
			err := utils.ModifyMCOAvailabilityConfig(testOptions, "Basic")
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())

		By("Checking MCO components in Basic mode")
		Eventually(func() error {
			err = utils.CheckMCOComponentsInBaiscMode(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("Modifying MCO availabilityConfig to enable high mode")
		Eventually(func() error {
			err = utils.ModifyMCOAvailabilityConfig(testOptions, "High")
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())

		By("Checking MCO components in High mode")
		Eventually(func() error {
			err = utils.CheckMCOComponentsInHighMode(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	}

	utils.PrintAllMCOPodsStatus(testOptions)
	utils.PrintAllOBAPodsStatus(testOptions)
	testFailed = testFailed || CurrentGinkgoTestDescription().Failed
}
