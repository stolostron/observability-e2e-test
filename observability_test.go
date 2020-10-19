package main_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

const (
	MCO_OPERATOR_NAMESPACE = "open-cluster-management"
	MCO_NAMESPACE          = "open-cluster-management-observability"
	MCO_LABEL              = "name=multicluster-observability-operator"
)

var (
	EventuallyTimeoutMinute  time.Duration = 60 * time.Second
	EventuallyIntervalSecond time.Duration = 1 * time.Second
)

var _ = Describe("testing all observability features", func() {
	var hubClient kubernetes.Interface
	BeforeEach(func() {
		By("connecting to the hub cluster")
		hubClient = utils.NewKubeClient(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext)
	})

	It("should have the expected running mco operator in namespace: open-cluster-management", func() {
		var podList, _ = hubClient.CoreV1().Pods(MCO_OPERATOR_NAMESPACE).List(metav1.ListOptions{LabelSelector: MCO_LABEL})
		Expect(len(podList.Items)).To(Equal(1))
		for _, pod := range podList.Items {
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
		}

		By("checking required CRDs have existed")
		Eventually(func() error {
			return utils.HaveCRDs(testOptions.HubCluster, testOptions.KubeConfig,
				[]string{
					"multiclusterobservabilities.observability.open-cluster-management.io",
					"observatoria.core.observatorium.io",
					"observabilityaddons.observability.open-cluster-management.io",
				})
		}).Should(Succeed())
	})

	It("should install mco instance sucessfully", func() {

		By("creating MCO namespace")
		Expect(utils.CreateMCONamespace(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)).NotTo(HaveOccurred())

		By("creating MCO pull secret")
		Expect(utils.CreatePullSecret(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)).NotTo(HaveOccurred())

		By("creating MCO object secret")
		Expect(utils.CreateObjSecret(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)).NotTo(HaveOccurred())

		By("creating MCO instance")
		mco := utils.NewMCOInstanceYaml("observability")
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, mco)).NotTo(HaveOccurred())
	})

	It("should have the expected running mco components in namespace: open-cluster-management-observability", func() {
		By("waiting for MCO deployments to be created")
		Eventually(func() error {
			return utils.HaveDeploymentsInNamespace(testOptions.HubCluster, testOptions.KubeConfig,
				MCO_NAMESPACE,
				[]string{
					"grafana",
					"observability-observatorium-observatorium-api",
					"observability-observatorium-thanos-query",
					"observability-observatorium-thanos-query-frontend",
					"observability-observatorium-thanos-receive-controller",
					"observatorium-operator",
					"rbac-query-proxy",
				})
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*10).Should(Succeed())

		Eventually(func() error {
			By("waiting for MCO statefulsets to be created")
			return utils.HaveStatefulSetsInNamespace(testOptions.HubCluster, testOptions.KubeConfig,
				MCO_NAMESPACE,
				[]string{
					"alertmanager",
					"observability-observatorium-thanos-compact",
					"observability-observatorium-thanos-receive-default",
					"observability-observatorium-thanos-rule",
					"observability-observatorium-thanos-store-memcached",
					"observability-observatorium-thanos-store-shard-0",
					"observability-observatorium-thanos-store-shard-1",
					"observability-observatorium-thanos-store-shard-2",
				})
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*10).Should(Succeed())
	})

	It("should uninstall MCO instance sucessfully", func() {
		By("uninstall MCO instance")
		err := utils.UninstallMCO(testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			By("waiting for delete all MCO components")
			var podList, _ = hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
			if len(podList.Items) != 0 {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*10).Should(Succeed())

		Eventually(func() error {
			By("waiting for delete MCO namespaces")
			err := hubClient.CoreV1().Namespaces().Delete(MCO_NAMESPACE, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
			return nil
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*10).Should(Succeed())

	})
})
