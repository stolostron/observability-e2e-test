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
	MCO_POD_NAMESPACE      = "open-cluster-management-observability"
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
})
