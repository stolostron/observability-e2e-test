package main_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

func uninstallMCO() {
	if os.Getenv("SKIP_UNINSTALL_STEP") == "true" {
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

	By("Uninstall MCO instance")
	err := utils.UninstallMCO(testOptions)
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for delete all MCO components")
	Eventually(func() error {
		var podList, _ = hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
		if len(podList.Items) != 0 {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Waiting for delete MCO addon instance")
	Eventually(func() error {
		gvr := utils.NewMCOAddonGVR()
		name := MCO_CR_NAME + "-addon"
		instance, _ := dynClient.Resource(gvr).Namespace("local-cluster").Get(name, metav1.GetOptions{})
		if instance != nil {
			return fmt.Errorf("Failed to delete MCO addon instance")
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Waiting for delete all MCO addon components")
	Eventually(func() error {
		var podList, _ = hubClient.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{})
		if len(podList.Items) != 0 {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Waiting for delete MCO namespaces")
	Eventually(func() error {
		err := hubClient.CoreV1().Namespaces().Delete(MCO_NAMESPACE, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
}
