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

	It("[P2][Sev2][Observability][Stable] Check compact args: --delete-delay=50h (retention/g0)", func() {
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
			return fmt.Errorf("Failed to check compact args: --delete-delay=50h. args is %v", argList)
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check store args: --ignore-deletion-marks-delay=25h (retention/g0)", func() {
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
			return fmt.Errorf("Failed to check store args: --ignore-deletion-marks-delay=25h. args is %v", argList)
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check receive args: --tsdb.retention=5d (retention/g0)", func() {
		Eventually(func() error {
			name := MCO_CR_NAME + "-thanos-receive-default"
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
			return fmt.Errorf("Failed to check receive args: --tsdb.retention=5d. args is %v", argList)
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check rule args: --tsdb.retention=5d (retention/g0)", func() {
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
			return fmt.Errorf("Failed to check rule args: --tsdb.retention=5d. args is %v", argList)
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check rule args: --tsdb.block-duration=3h (retention/g0)", func() {
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
			return fmt.Errorf("Failed to check rule args: --tsdb.block-duration=3h. args is %v", argList)
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
