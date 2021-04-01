// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
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
	for _, pod := range podList.Items {
		Expect(string(pod.Status.Phase)).To(Equal("Running"))
	}

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
	Expect(utils.CreatePullSecret(testOptions)).NotTo(HaveOccurred())
	Expect(utils.CreateObjSecret(testOptions)).NotTo(HaveOccurred())
	//set resource quota and limit range for canary environment to avoid destruct the node
	yamlB, err := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/policy"})
	Expect(err).NotTo(HaveOccurred())
	Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

	By("Creating MCO instance of v1beta1")
	v1beta1KustomizationPath := "../../observability-gitops/mco/e2e/v1beta1"
	yamlB, err = kustomize.Render(kustomize.Options{KustomizationPath: v1beta1KustomizationPath})
	Expect(err).NotTo(HaveOccurred())
	Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

	By("Waiting for MCO ready status")
	allPodsIsReady := false
	Eventually(func() bool {
		instance, err := dynClient.Resource(utils.NewMCOGVRV1BETA1()).Get(MCO_CR_NAME, metav1.GetOptions{})
		if err == nil {
			allPodsIsReady = utils.StatusContainsTypeEqualTo(instance, "Ready")
			return allPodsIsReady
		}
		return false
	}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(BeTrue())

	if !allPodsIsReady {
		utils.PrintAllMCOPodsStatus(testOptions)
	}

	By("Check clustermanagementaddon CR is created")
	Eventually(func() error {
		_, err := dynClient.Resource(utils.NewMCOClusterManagementAddonsGVR()).Get("observability-controller", metav1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}).Should(Succeed())

	By("Check the api conversion is working as expected")
	v1beta1Tov1beta2GoldenPath := "../../observability-gitops/mco/e2e/v1beta1/observability-v1beta1-to-v1beta2-golden.yaml"
	err = utils.CheckMCOConversion(testOptions, v1beta1Tov1beta2GoldenPath)
	Expect(err).NotTo(HaveOccurred())

	By("Apply MCO instance of v1beta2")
	v1beta2KustomizationPath := "../../observability-gitops/mco/e2e/v1beta2"
	yamlB, err = kustomize.Render(kustomize.Options{KustomizationPath: v1beta2KustomizationPath})
	Expect(err).NotTo(HaveOccurred())
	Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

	By("Waiting for MCO ready status")
	allPodsIsReady = false
	Eventually(func() bool {
		instance, err := dynClient.Resource(utils.NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
		if err == nil {
			allPodsIsReady = utils.StatusContainsTypeEqualTo(instance, "Ready")
			return allPodsIsReady
		}
		return false
	}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(BeTrue())

	if !allPodsIsReady {
		utils.PrintAllMCOPodsStatus(testOptions)
	}

	By("Check clustermanagementaddon CR is created")
	Eventually(func() error {
		_, err := dynClient.Resource(utils.NewMCOClusterManagementAddonsGVR()).Get("observability-controller", metav1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}).Should(Succeed())
}
