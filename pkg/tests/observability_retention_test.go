// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"fmt"

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

	It("[P2][Sev2][Observability][Integration] Checking retention config in components args (retention/g0)", func() {
		By("check compact args: --delete-delay=50h")
		Eventually(func() error {
			name := MCO_CR_NAME + "-thanos-compact"
			compact, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			argList := compact.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--delete-delay=50h" {
					return nil
				}
			}
			return fmt.Errorf("Failed to check compact args: --delete-delay=50h")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("check store args: --ignore-deletion-marks-delay=25h")
		Eventually(func() error {
			name := MCO_CR_NAME + "-thanos-store-shard-0"
			store, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			argList := store.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--ignore-deletion-marks-delay=25h" {
					return nil
				}
			}
			return fmt.Errorf("Failed to check store args: --ignore-deletion-marks-delay=25h")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("check receive args: --tsdb.retention=5d")
		Eventually(func() error {
			name := MCO_CR_NAME + "thanos-receive-default"
			receive, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			argList := receive.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--tsdb.retention=5d" {
					return nil
				}
			}
			return fmt.Errorf("Failed to check receive args: --tsdb.retention=5d")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("check rule args: --tsdb.retention=5d")
		Eventually(func() error {
			name := MCO_CR_NAME + "-thanos-rule"
			rule, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			argList := rule.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--tsdb.retention=5d" {
					return nil
				}
			}
			return fmt.Errorf("Failed to check rule args: --tsdb.retention=5d")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		By("check rule args: --tsdb.block-duration=3h")
		Eventually(func() error {
			name := MCO_CR_NAME + "-thanos-rule"
			rule, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			argList := rule.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--tsdb.block-duration=3h" {
					return nil
				}
			}
			return fmt.Errorf("Failed to check rule args: --tsdb.block-duration=3h")
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	AfterEach(func() {
		if testFailed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		} else {
			Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
