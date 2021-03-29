// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

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
		It("Updating observatorium cr (spec.rule.replicas) should be automatically reverted", func() {
			crName := "observability"
			oldResourceVersion := ""
			updateReplicas := int64(2)
			Eventually(func() error {
				cr, err := dynClient.Resource(utils.NewMCOMObservatoriumGVR()).Namespace(MCO_NAMESPACE).Get(crName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				cr.Object["spec"].(map[string]interface{})["thanos"].(map[string]interface{})["rule"].(map[string]interface{})["replicas"] = updateReplicas
				oldResourceVersion = cr.Object["metadata"].(map[string]interface{})["resourceVersion"].(string)
				_, err = dynClient.Resource(utils.NewMCOMObservatoriumGVR()).Namespace(MCO_NAMESPACE).Update(cr, metav1.UpdateOptions{})
				return err
			}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(Succeed())

			Eventually(func() bool {
				cr, err := dynClient.Resource(utils.NewMCOMObservatoriumGVR()).Namespace(MCO_NAMESPACE).Get(crName, metav1.GetOptions{})
				if err == nil {
					replicasNew := cr.Object["spec"].(map[string]interface{})["thanos"].(map[string]interface{})["rule"].(map[string]interface{})["replicas"].(int64)
					newResourceVersion := cr.Object["metadata"].(map[string]interface{})["resourceVersion"].(string)
					if newResourceVersion != oldResourceVersion &&
						replicasNew != updateReplicas {
						return true
					}
				}
				return false
			}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*1).Should(BeTrue())
		})
	})

	AfterEach(func() {
		if testFailed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
