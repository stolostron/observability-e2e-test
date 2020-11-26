package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

const (
	dashboardName   = "sample-dashboard"
	dashboardTitile = "Sample Dashboard for E2E"
)

func getSampleDashboardConfigmap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboardName,
			Namespace: MCO_NAMESPACE,
			Labels: map[string]string{
				"grafana-custom-dashboard": "true",
			},
		},
		Data: map[string]string{"data": `
{
	"id": "e2e",
	"uid": null,
	"title": "Sample Dashboard for E2E",
	"tags": [ "test" ],
	"timezone": "browser",
	"schemaVersion": 16,
	"version": 1
	}
`},
	}
}

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

	It("should have custom dashboard which defined in configmap (dashboard/g0)", func() {
		By("Creating custom dashboard configmap")
		err = utils.CreateConfigMap(testOptions, true, getSampleDashboardConfigmap())
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			_, result := utils.ContainDashboard(testOptions, dashboardTitile)
			return result
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("should have no custom dashboard in grafana after related configmap removed(dashboard/g0)", func() {
		By("Deleting custom dashboard configmap")
		err = utils.DeleteConfigMap(testOptions, true, dashboardName, MCO_NAMESPACE)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			_, result := utils.ContainDashboard(testOptions, dashboardTitile)
			return result
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(BeFalse())
	})
})
