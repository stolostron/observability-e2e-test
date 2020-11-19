package main_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management/observability-e2e-test/utils"
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

	It("[P1,Sev1,observability] should have metrics collector pod restart if cert secret re-generated (certrenew/g0)", func() {
		By("Waiting for pods ready: observability-observatorium-observatorium-api, rbac-query-proxy, metrics-collector-deployment")
		collectorPodName := ""
		apiPodName := ""
		rbacPodName := ""
		Eventually(func() bool {
			if collectorPodName == "" {
				_, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
				if podList != nil && len(podList.Items) > 0 {
					collectorPodName = podList.Items[0].Name
				}
			}
			if apiPodName == "" {
				_, podList := utils.GetPodList(testOptions, true, MCO_ADDON_NAMESPACE, "app.kubernetes.io/name=observatorium-api")
				if podList != nil && len(podList.Items) > 0 {
					apiPodName = podList.Items[0].Name
				}
			}
			if rbacPodName == "" {
				_, podList := utils.GetPodList(testOptions, true, MCO_ADDON_NAMESPACE, "app=rbac-query-proxy")
				if podList != nil && len(podList.Items) > 0 {
					rbacPodName = podList.Items[0].Name
				}
			}
			if collectorPodName == "" && apiPodName == "" && rbacPodName == "" {
				return false
			}
			return true
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Deleting certificate secret to simulate certificate renew")
		err := utils.DeleteCertSecret(testOptions)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("Waiting for old pod removed: %s", rbacPodName))
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, true, MCO_ADDON_NAMESPACE, "app=rbac-query-proxy")
			if err == nil {
				for _, pod := range podList.Items {
					if pod.Name == rbacPodName {
						return true
					}
				}
			} else {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeFalse())

		By(fmt.Sprintf("Waiting for old pod removed: %s", apiPodName))
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if err == nil {
				for _, pod := range podList.Items {
					if pod.Name == apiPodName {
						return true
					}
				}
			} else {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeFalse())

		By(fmt.Sprintf("Waiting for old pod removed: %s", collectorPodName))
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if err == nil {
				for _, pod := range podList.Items {
					if pod.Name == collectorPodName {
						return true
					}
				}
			} else {
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeFalse())
	})
})
