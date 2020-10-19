package main_test

import (
	"fmt"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-cluster-management/observability-e2e-test/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	OBSERVABILITY_POD_NAMESPACE = "open-cluster-management"
	OBSERVABILITY_DEPLOY_PREFIX = "multicluster-observability-operator"
	OBSERVABILITY_LABEL         = "name=multicluster-observability-operator"
)

var _ = Describe("MCO Operator testing", func() {
	var hubClient kubernetes.Interface
	BeforeEach(func() {
		io.WriteString(GinkgoWriter, fmt.Sprintf("\n\nConnecting to the Hub with master-url: %s\n\tcontext: %s\n\tfrom kubeconfig: %s\n\n", testOptions.HubCluster.MasterURL, testOptions.HubCluster.KubeContext, testOptions.KubeConfig))
		hubClient = utils.NewKubeClient(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext)
	})

	It("should have the expected running mco operator in namespace: open-cluster-management (ingress/g0)", func() {
		var podList, _ = hubClient.CoreV1().Pods(OBSERVABILITY_POD_NAMESPACE).List(metav1.ListOptions{LabelSelector: OBSERVABILITY_LABEL})
		Expect(len(podList.Items)).To(Equal(1))
		for _, pod := range podList.Items {
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
		}
	})
})
