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

	By("Creating MCO instance")
	mco := utils.NewMCOInstanceYaml(MCO_CR_NAME)
	Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, mco)).NotTo(HaveOccurred())

	By("Waiting for MCO ready status")
	allPodsIsReady := false
	Eventually(func() bool {
		instance, err := dynClient.Resource(utils.NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
		if err == nil {
			allPodsIsReady = utils.StatusContainsTypeEqualTo(instance, "Ready")
			return allPodsIsReady
		}
		return false
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

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

	By("Checking metrics default values on managed cluster")
	mco_res, err := dynClient.Resource(utils.NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	observabilityAddonSpec := mco_res.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})
	Expect(observabilityAddonSpec["enableMetrics"]).To(Equal(true))
	Expect(observabilityAddonSpec["interval"]).To(Equal(int64(60)))

	By("Checking pvc and storageclass is the default")
	mco_sc, err := dynClient.Resource(utils.NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	spec := mco_sc.Object["spec"].(map[string]interface{})
	sizeInCR := spec["storageConfigObject"].(map[string]interface{})["statefulSetSize"].(string)
	scInCR := spec["storageConfigObject"].(map[string]interface{})["statefulSetStorageClass"].(string)

	scList, err := hubClient.StorageV1().StorageClasses().List(metav1.ListOptions{})
	scMatch := false
	defaultSC := ""
	for _, sc := range scList.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			defaultSC = sc.Name
		}
		if sc.Name == scInCR {
			scMatch = true
		}
	}
	expectedSC := defaultSC
	if scMatch {
		expectedSC = scInCR
	}

	Eventually(func() error {
		pvcList, err := hubClient.CoreV1().PersistentVolumeClaims(MCO_NAMESPACE).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, pvc := range pvcList.Items {
			pvcSize := pvc.Spec.Resources.Requests["storage"]
			scName := *pvc.Spec.StorageClassName
			statusPhase := pvc.Status.Phase
			if pvcSize.String() != sizeInCR || scName != expectedSC || statusPhase != "Bound" {
				return fmt.Errorf("PVC check failed, pvcSize = %s, sizeInCR = %s, scName = %s, expectedSC = %s, statusPhase = %s", pvcSize.String(), sizeInCR, scName, expectedSC, statusPhase)
			}
		}
		return nil
	}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(Succeed())
}
