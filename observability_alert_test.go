package main_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"

	// . "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	It("should have the expected stateful sets (alert/g0)", func() {
		By("Checking if STS: Alertmanager and observability-observatorium-thanos-rule is existed")
		statefulSetList, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).List(metav1.ListOptions{})
		fmt.Print(len(statefulSetList.Items))
		for _, sts := range statefulSetList.Items {
			fmt.Println("Statefulset", sts, err)
		}
	})

	It("should have the expected configmap (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-default-rules is existed")
		configMapList, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).List(metav1.ListOptions{})
		fmt.Print(len(configMapList.Items))
		for _, cm := range configMapList.Items {
			fmt.Println("Configmap", cm, err)
		}
	})

	It("should have the expected secret (alert/g0)", func() {
		By("Checking if SECRETS: alertmanager-config is existed")
		secretList, err := hubClient.CoreV1().Secrets(MCO_NAMESPACE).List(metav1.ListOptions{})
		fmt.Print(len(secretList.Items))
		for _, secret := range secretList.Items {
			fmt.Println("Secret", secret, err)
		}
	})
})
