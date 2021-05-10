// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

func installMCO() {
	if os.Getenv("SKIP_INSTALL_STEP") == "true" {
		return
	}

	hubClient := utils.NewKubeClient(
		testOptions.HubCluster.MasterURL,
		testOptions.KubeConfig,
		testOptions.HubCluster.KubeContext)

	dynClient := utils.NewKubeClientDynamic(
		testOptions.HubCluster.MasterURL,
		testOptions.KubeConfig,
		testOptions.HubCluster.KubeContext)

	By("Checking MCO operator is existed")
	podList, err := hubClient.CoreV1().Pods(MCO_OPERATOR_NAMESPACE).List(metav1.ListOptions{LabelSelector: MCO_LABEL})
	Expect(len(podList.Items)).To(Equal(1))
	Expect(err).NotTo(HaveOccurred())
	mcoPod := ""
	for _, pod := range podList.Items {
		mcoPod = pod.GetName()
		Expect(string(mcoPod)).NotTo(Equal(""))
		Expect(string(pod.Status.Phase)).To(Equal("Running"))
	}

	// print mco logs if MCO installation failed
	defer func(testOptions utils.TestOptions, isHub bool, namespace, podName, containerName string, previous bool, tailLines int64) {
		if testFailed {
			mcoLogs, err := utils.GetPodLogs(testOptions, isHub, namespace, podName, containerName, previous, tailLines)
			Expect(err).NotTo(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "[DEBUG] MCO is installed failed, checking MCO operator logs: %s\n", mcoLogs)
		} else {
			fmt.Fprintf(GinkgoWriter, "[DEBUG] MCO is installed successfully!")
		}
	}(testOptions, false, MCO_OPERATOR_NAMESPACE, mcoPod, "multicluster-observability-operator", false, 1000)

	By("Checking Required CRDs is existed")
	Eventually(func() error {
		return utils.HaveCRDs(testOptions.HubCluster, testOptions.KubeConfig,
			[]string{
				"multiclusterobservabilities.observability.open-cluster-management.io",
				"observatoria.core.observatorium.io",
				"observabilityaddons.observability.open-cluster-management.io",
			})
	}).Should(Succeed())

	Expect(utils.CreateMCONamespace(testOptions)).NotTo(HaveOccurred())
	if os.Getenv("IS_CANARY_ENV") == "true" {
		Expect(utils.CreatePullSecret(testOptions)).NotTo(HaveOccurred())
		Expect(utils.CreateObjSecret(testOptions)).NotTo(HaveOccurred())
	}
	//set resource quota and limit range for canary environment to avoid destruct the node
	yamlB, err := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/policy"})
	Expect(err).NotTo(HaveOccurred())
	Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

	if os.Getenv("IS_CANARY_ENV") != "true" {
		By("Creating the MCO testing RBAC resources")
		Expect(utils.CreateMCOTestingRBAC(testOptions)).NotTo(HaveOccurred())
	}

	if os.Getenv("SKIP_INTEGRATION_CASES") != "true" {
		By("Creating MCO instance of v1beta1")
		v1beta1KustomizationPath := "../../observability-gitops/mco/e2e/v1beta1"
		yamlB, err = kustomize.Render(kustomize.Options{KustomizationPath: v1beta1KustomizationPath})
		Expect(err).NotTo(HaveOccurred())
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		By("Waiting for MCO ready status")
		allPodsIsReady := false
		Eventually(func() error {
			instance, err := dynClient.Resource(utils.NewMCOGVRV1BETA1()).Get(MCO_CR_NAME, metav1.GetOptions{})
			if err == nil {
				allPodsIsReady = utils.StatusContainsTypeEqualTo(instance, "Ready")
				if allPodsIsReady {
					testFailed = false
					return nil
				}
			}
			testFailed = true
			if instance != nil && instance.Object != nil {
				return fmt.Errorf("MCO componnets cannot be running in 20 minutes. check the MCO CR status for the details: %v", instance.Object["status"])
			} else {
				return fmt.Errorf("Wait for reconciling.")
			}
		}, EventuallyTimeoutMinute*20, EventuallyIntervalSecond*5).Should(Succeed())

		By("Check clustermanagementaddon CR is created")
		Eventually(func() error {
			_, err := dynClient.Resource(utils.NewMCOClusterManagementAddonsGVR()).Get("observability-controller", metav1.GetOptions{})
			if err != nil {
				testFailed = true
				return err
			}
			testFailed = false
			return nil
		}).Should(Succeed())

		By("Check the api conversion is working as expected")
		v1beta1Tov1beta2GoldenPath := "../../observability-gitops/mco/e2e/v1beta1/observability-v1beta1-to-v1beta2-golden.yaml"
		err = utils.CheckMCOConversion(testOptions, v1beta1Tov1beta2GoldenPath)
		Expect(err).NotTo(HaveOccurred())
	}

	By("Apply MCO instance of v1beta2")
	v1beta2KustomizationPath := "../../observability-gitops/mco/e2e/v1beta2"
	yamlB, err = kustomize.Render(kustomize.Options{KustomizationPath: v1beta2KustomizationPath})
	Expect(err).NotTo(HaveOccurred())
	Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

	By("Waiting for MCO ready status")
	allPodsIsReady := false
	Eventually(func() error {
		err = utils.CheckMCOComponentsInHighMode(testOptions)
		if err != nil {
			testFailed = true
			return err
		}
		allPodsIsReady = true
		testFailed = false
		return nil
	}, EventuallyTimeoutMinute*25, EventuallyIntervalSecond*5).Should(Succeed())

	if !allPodsIsReady {
		utils.PrintAllMCOPodsStatus(testOptions)
	}

	By("Checking placementrule CR is created")
	Eventually(func() error {
		_, err := dynClient.Resource(utils.NewOCMPlacementRuleGVR()).Namespace(utils.MCO_NAMESPACE).Get("observability", metav1.GetOptions{})
		if err != nil {
			testFailed = true
			return err
		}
		testFailed = false
		return nil
	}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())

	if os.Getenv("IS_CANARY_ENV") != "true" {
		// TODO(morvencao): remove the patch from placement is implemented by server foundation.
		By("Patching the placementrule CR's status")
		token, err := utils.FetchBearerToken(testOptions)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			err = utils.PatchPlacementRule(testOptions, token)
			if err != nil {
				testFailed = true
				return err
			}
			testFailed = false
			return nil
		}).Should(Succeed())

		By("Waiting for MCO addon components ready")
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if len(podList.Items) == 1 && err == nil {
				testFailed = false
				return true
			}
			testFailed = true
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	}

	By("Check clustermanagementaddon CR is created")
	Eventually(func() error {
		_, err := dynClient.Resource(utils.NewMCOClusterManagementAddonsGVR()).Get("observability-controller", metav1.GetOptions{})
		if err != nil {
			testFailed = true
			return err
		}
		testFailed = false
		return nil
	}).Should(Succeed())
}
