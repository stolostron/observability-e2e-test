package main_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

const (
	MCO_OPERATOR_NAMESPACE = "open-cluster-management"
	MCO_CR_NAME            = "observability"
	MCO_NAMESPACE          = "open-cluster-management-observability"
	MCO_ADDON_NAMESPACE    = "open-cluster-management-addon-observability"
	MCO_LABEL              = "name=multicluster-observability-operator"
)

var (
	EventuallyTimeoutMinute  time.Duration = 60 * time.Second
	EventuallyIntervalSecond time.Duration = 1 * time.Second

	hubClient kubernetes.Interface
	dynClient dynamic.Interface
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

	It("should have the expected args in compact pod (reconcile/g0)", func() {
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
			return errors.New("Failed to find modified retention field")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("should have node selector: kubernetes.io/os=linux (reconcile/g0)", func() {
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
	})

	It("should work in basic mode (reconcile/g0)", func() {
		By("Modifying MCO availabilityConfig to enable basic mode")
		err := utils.ModifyMCOAvailabilityConfig(testOptions, "Basic")
		Expect(err).ToNot(HaveOccurred())

		By("Checking MCO components in Basic mode")
		Eventually(func() error {
			err = utils.CheckMCOComponentsInBaiscMode(testOptions)
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("Modifying MCO availabilityConfig to enable high mode")
		err = utils.ModifyMCOAvailabilityConfig(testOptions, "High")
		Expect(err).ToNot(HaveOccurred())
	})
})
