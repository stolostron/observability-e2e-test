// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stolostron/observability-e2e-test/pkg/utils"
)

var _ = Describe("Observability:", func() {

	var (
		deleteDelay              = "48h"
		retentionInLocal         = "24h"
		blockDuration            = "2h"
		ignoreDeletionMarksDelay = "24h"
	)

	BeforeEach(func() {
		hubClient = utils.NewKubeClient(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)
		dynClient = utils.NewKubeClientDynamic(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)

		mcoRes, err := dynClient.Resource(utils.NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}

		if _, adv := mcoRes.Object["spec"].(map[string]interface{})["advanced"]; adv {
			if _, rec := mcoRes.Object["spec"].(map[string]interface{})["advanced"].(map[string]interface{})["retentionConfig"]; rec {
				for k, v := range mcoRes.Object["spec"].(map[string]interface{})["advanced"].(map[string]interface{})["retentionConfig"].(map[string]interface{}) {
					switch k {
					case "deleteDelay":
						deleteDelay = reflect.ValueOf(v).String()
						idmk, _ := strconv.Atoi(deleteDelay[:len(deleteDelay)-1])
						ignoreDeletionMarksDelay = fmt.Sprintf("%.f", math.Ceil(float64(idmk)/float64(2))) + deleteDelay[len(deleteDelay)-1:]
					case "retentionInLocal":
						retentionInLocal = reflect.ValueOf(v).String()
					case "blockDuration":
						blockDuration = reflect.ValueOf(v).String()
					}
				}
			}
		}
	})

	It("[P2][Sev2][Observability][Stable] Check and tune backup retention settings in MCO CR - Check compact args (retention/g0):", func() {
		By("--delete-delay=" + deleteDelay)
		Eventually(func() error {
			compacts, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).List(metav1.ListOptions{
				LabelSelector: THANOS_COMPACT_LABEL,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(compacts.Items)).NotTo(Equal(0))

			argList := (*compacts).Items[0].Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--delete-delay="+deleteDelay {
					return nil
				}
			}
			return fmt.Errorf("Failed to check compact args: --delete-delay="+deleteDelay+". args is %v", argList)
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check and tune backup retention settings in MCO CR - Check store args (retention/g0):", func() {
		By("--ignore-deletion-marks-delay=" + ignoreDeletionMarksDelay)
		Eventually(func() error {
			stores, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).List(metav1.ListOptions{
				LabelSelector: THANOS_STORE_LABEL,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(stores.Items)).NotTo(Equal(0))

			argList := (*stores).Items[0].Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--ignore-deletion-marks-delay="+ignoreDeletionMarksDelay {
					return nil
				}
			}
			return fmt.Errorf("Failed to check store args: --ignore-deletion-marks-delay="+ignoreDeletionMarksDelay+". The args is: %v", argList)
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check and tune backup retention settings in MCO CR - Check receive args (retention/g0):", func() {
		By("--tsdb.retention=" + retentionInLocal)
		Eventually(func() error {
			receives, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).List(metav1.ListOptions{
				LabelSelector: THANOS_RECEIVE_LABEL,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(receives.Items)).NotTo(Equal(0))

			argList := (*receives).Items[0].Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--tsdb.retention="+retentionInLocal {
					return nil
				}
			}
			return fmt.Errorf("Failed to check receive args: --tsdb.retention="+retentionInLocal+". The args is: %v", argList)
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check and tune backup retention settings in MCO CR - Check rule args (retention/g0):", func() {
		By("--tsdb.retention=" + retentionInLocal)
		Eventually(func() error {
			rules, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).List(metav1.ListOptions{
				LabelSelector: THANOS_RULE_LABEL,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules.Items)).NotTo(Equal(0))

			argList := (*rules).Items[0].Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--tsdb.retention="+retentionInLocal {
					return nil
				}
			}
			return fmt.Errorf("Failed to check rule args: --tsdb.retention="+retentionInLocal+". The args is: %v", argList)
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2][Sev2][Observability][Stable] Check and tune backup retention settings in MCO CR - Check rule args (retention/g0):", func() {
		By("--tsdb.block-duration=" + blockDuration)
		Eventually(func() error {
			rules, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).List(metav1.ListOptions{
				LabelSelector: THANOS_RULE_LABEL,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules.Items)).NotTo(Equal(0))

			argList := (*rules).Items[0].Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--tsdb.block-duration="+blockDuration {
					return nil
				}
			}
			return fmt.Errorf("Failed to check rule args: --tsdb.block-duration="+blockDuration+". The args is: %v", argList)
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())
	})

	JustAfterEach(func() {
		Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
