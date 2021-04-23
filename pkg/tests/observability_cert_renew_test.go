// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog"

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

	It("[P1][Sev1][Observability][Integration] Should have metrics collector pod restart if cert secret re-generated (certrenew/g0)", func() {
		Skip("[P1][Sev1][Observability] Should have metrics collector pod restart if cert secret re-generated (certrenew/g0)")
		By("Waiting for pods ready: observability-observatorium-api, metrics-collector-deployment")
		collectorPodName := ""
		apiPodsName := map[string]bool{}
		Eventually(func() bool {
			if collectorPodName == "" {
				_, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
				if podList != nil && len(podList.Items) > 0 {
					collectorPodName = podList.Items[0].Name
				}
			}
			if len(apiPodsName) == 0 {
				_, podList := utils.GetPodList(testOptions, true, MCO_NAMESPACE, "app.kubernetes.io/name=observatorium-api")
				if podList != nil {
					for _, pod := range podList.Items {
						apiPodsName[pod.Name] = false
					}
				}

			}
			if collectorPodName == "" && len(apiPodsName) == 0 {
				return false
			}
			return true
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Deleting certificate secret to simulate certificate renew")
		err := utils.DeleteCertSecret(testOptions)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("Waiting for old pods removed: %v and new pods created", apiPodsName))
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, true, MCO_NAMESPACE, "app.kubernetes.io/name=observatorium-api")
			if err == nil {
				if len(podList.Items) != 0 {
					for oldPodName := range apiPodsName {
						apiPodsName[oldPodName] = true
						for _, pod := range podList.Items {
							if oldPodName == pod.Name {
								apiPodsName[oldPodName] = false
							}
						}
					}
				}
				allRecreated := true
				for _, value := range apiPodsName {
					if !value {
						allRecreated = false
					}
				}
				if allRecreated {
					return true
				}
			} else {
				return false
			}

			// debug code to check label "certmanager.k8s.io/time-restarted"
			err, deployment := utils.GetDeployment(testOptions, true, MCO_CR_NAME+"-observatorium-api", MCO_NAMESPACE)
			if err == nil {
				klog.V(1).Infof("labels: <%v>", deployment.ObjectMeta.Labels)
			}

			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By(fmt.Sprintf("Waiting for old pod removed: %s and new pod created", collectorPodName))
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if err == nil {
				for _, pod := range podList.Items {
					if pod.Name != collectorPodName {
						return true
					}
				}
			} else {
				return false
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	AfterEach(func() {
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
		if testFailed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		} else {
			Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
		}
	})
})
