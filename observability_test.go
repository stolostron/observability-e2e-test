package main_test

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

const (
	MCO_OPERATOR_NAMESPACE = "open-cluster-management"
	MCO_CR_NAME            = "observability"
	MCO_NAMESPACE          = "open-cluster-management-observability"
	MCO_ADDON_NAMESPACE    = "open-cluster-management-addon-observability"
	MCO_LABEL              = "name=multicluster-observability-operator"
)

var (
	EventuallyTimeoutMinute  time.Duration = 60 * time.Second
	EventuallyIntervalSecond time.Duration = 1 * time.Second

	hubClient kubernetes.Interface
	dynClient dynamic.Interface
)

var _ = Describe("Observability", func() {
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

	It("Observability: MCO Operator is created", func() {
		var podList, _ = hubClient.CoreV1().Pods(MCO_OPERATOR_NAMESPACE).List(metav1.ListOptions{LabelSelector: MCO_LABEL})
		Expect(len(podList.Items)).To(Equal(1))
		for _, pod := range podList.Items {
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
		}
	})

	It("Observability: Required CRDs are created", func() {
		Eventually(func() error {
			return utils.HaveCRDs(testOptions.HubCluster, testOptions.KubeConfig,
				[]string{
					"multiclusterobservabilities.observability.open-cluster-management.io",
					"observatoria.core.observatorium.io",
					"observabilityaddons.observability.open-cluster-management.io",
				})
		}).Should(Succeed())
	})

	It("Observability: All required components are deployed and running", func() {
		Expect(utils.CreateMCONamespace(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)).NotTo(HaveOccurred())

		Expect(utils.CreatePullSecret(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)).NotTo(HaveOccurred())

		Expect(utils.CreateObjSecret(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)).NotTo(HaveOccurred())

		By("Creating MCO instance")
		mco := utils.NewMCOInstanceYaml("observability")
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, mco)).NotTo(HaveOccurred())

		By("Waiting for MCO ready status")
		Eventually(func() bool {
			instance, err := dynClient.Resource(utils.NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
			if err == nil {
				return utils.StatusContainsTypeEqualTo(instance, "Ready")
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("Observability: Grafana console can be accessible", func() {
		Eventually(func() error {
			config, err := utils.LoadConfig(testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)
			if err != nil {
				return err
			}

			req, err := http.NewRequest("GET", "https://multicloud-console.apps."+baseDomain+"/grafana/", nil)
			if err != nil {
				return err
			}

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			req.Header.Set("Authorization", "Bearer "+config.BearerToken)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return errors.New("Failed to access grafana console")
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("Observability: retentionResolutionRaw is modified", func() {
		Eventually(func() error {
			By("Modifying MCO retentionResolutionRaw filed")
			err := utils.ModifyMCORetentionResolutionRaw(
				testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)
			if err != nil {
				return err
			}

			By("Waiting for MCO retentionResolutionRaw filed to take effect")
			compact, getError := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get("observability-observatorium-thanos-compact", metav1.GetOptions{})
			if getError != nil {
				return getError
			}
			argList := compact.Spec.Template.Spec.Containers[0].Args
			for _, arg := range argList {
				if arg == "--retention.resolution-raw=3d" {
					return nil
				}
			}
			return errors.New("Failed to find modified retention field")

		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("Observability: Managed cluster metrics shows up in Grafana console", func() {
		Eventually(func() error {

			By("Should have metric data in Grafana console")
			config, err := utils.LoadConfig(testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)
			if err != nil {
				return err
			}
			path := "/grafana/api/datasources/proxy/1/api/v1/"
			queryParams := "query?query=cluster%3Acapacity_cpu_cores%3Asum"
			req, err := http.NewRequest(
				"GET",
				"https://multicloud-console.apps."+baseDomain+path+queryParams,
				nil)

			if err != nil {
				return err
			}

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			req.Header.Set("Authorization", "Bearer "+config.BearerToken)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return errors.New("Failed to access managed cluster metrics via grafana console")
			}

			metricResult, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if !strings.Contains(string(metricResult), `status":"success"`) {
				return errors.New("Failed to find valid status from response")
			}

			if !strings.Contains(string(metricResult), `"__name__":"cluster:capacity_cpu_cores:sum"`) {
				return errors.New("Failed to find metric name from response")
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("Observability: Modify availabilityConfig from High to Basic", func() {
		Eventually(func() error {
			By("Modifying MCO availabilityConfig filed")
			err := utils.ModifyMCOAvailabilityConfig(
				testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)
			if err != nil {
				return err
			}

			By("Checking MCO components in Basic mode")
			err = utils.CheckMCOComponentsInBaiscMode(
				testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)

			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("Observability: disable observabilityaddon", func() {
		Eventually(func() error {
			By("Modifying MCO observabilityAddonSpec.enableMetrics filed")
			err := utils.ModifyMCOobservabilityAddonSpec(
				testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)
			if err != nil {
				return err
			}

			By("Waiting for MCO addon components disapear")
			addonLabel := "component=metrics-collector"
			var podList, _ = hubClient.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{LabelSelector: addonLabel})
			if len(podList.Items) != 0 {
				return errors.New("Failed to disable observability addon")
			}

			By("Should have not metric data in Grafana console")
			config, err := utils.LoadConfig(testOptions.HubCluster.MasterURL,
				testOptions.KubeConfig,
				testOptions.HubCluster.KubeContext)
			if err != nil {
				return err
			}
			path := "/grafana/api/datasources/proxy/1/api/v1/"
			queryParams := "query?query=cluster%3Acapacity_cpu_cores%3Asum"
			req, err := http.NewRequest(
				"GET",
				"https://multicloud-console.apps."+baseDomain+path+queryParams,
				nil)

			if err != nil {
				return err
			}

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			req.Header.Set("Authorization", "Bearer "+config.BearerToken)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return errors.New("Failed to access managed cluster metrics via grafana console")
			}

			metricResult, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if !strings.Contains(string(metricResult), `status":"success"`) {
				return errors.New("Failed to find valid status from response")
			}

			if strings.Contains(string(metricResult), `"__name__":"cluster:capacity_cpu_cores:sum"`) {
				return errors.New("Found metric name from response")
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("Observability: Clean up", func() {
		By("Uninstall MCO instance")
		err := utils.UninstallMCO(testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			By("Waiting for delete all MCO components")
			var podList, _ = hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
			if len(podList.Items) != 0 {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		Eventually(func() error {
			By("Waiting for delete MCO addon instance")
			gvr := utils.NewMCOAddonGVR()
			instance, _ := dynClient.Resource(gvr).Namespace("local-cluster").Get("observability-addon", metav1.GetOptions{})
			if instance != nil {
				return errors.New("Failed to delete MCO addon instance")
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		Eventually(func() error {
			By("Waiting for delete all MCO addon components")
			var podList, _ = hubClient.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{})
			if len(podList.Items) != 0 {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

		Eventually(func() error {
			By("Waiting for delete MCO namespaces")
			err := hubClient.CoreV1().Namespaces().Delete(MCO_NAMESPACE, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})
})
