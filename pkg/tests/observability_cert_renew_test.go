package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
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

	It("[P1][Sev1][Observability] Should have metrics collector pod restart if cert secret re-generated (certrenew/g0)", func() {
		By("Waiting for pods ready: observability-observatorium-observatorium-api, rbac-query-proxy, metrics-collector-deployment")
		collectorPodName := ""
		hubPodsName := []string{}
		Eventually(func() bool {
			if collectorPodName == "" {
				_, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
				if podList != nil && len(podList.Items) > 0 {
					collectorPodName = podList.Items[0].Name
				}
			}
			if collectorPodName == "" {
				return false
			}
			hubPodsName = []string{}
			_, apiPodList := utils.GetPodList(testOptions, true, MCO_NAMESPACE, "app.kubernetes.io/name=observatorium-api")
			if apiPodList != nil && len(apiPodList.Items) != 0 {
				for _, pod := range apiPodList.Items {
					hubPodsName = append(hubPodsName, pod.Name)
				}
			} else {
				return false
			}
			_, rbacPodList := utils.GetPodList(testOptions, true, MCO_NAMESPACE, "app=rbac-query-proxy")
			if rbacPodList != nil && len(rbacPodList.Items) != 0 {
				for _, pod := range rbacPodList.Items {
					hubPodsName = append(hubPodsName, pod.Name)
				}
			} else {
				return false
			}

			return true
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Deleting certificate secret to simulate certificate renew")
		err := utils.DeleteCertSecret(testOptions)
		Expect(err).ToNot(HaveOccurred())

		if !utils.IsCanaryEnvironment(testOptions) {
			//When a secret currently consumed in a volume is updated, projected keys are eventually updated as well. The kubelet checks whether the mounted secret is fresh on every periodic sync. However, the kubelet uses its local cache for getting the current value of the Secret. The type of the cache is configurable using the ConfigMapAndSecretChangeDetectionStrategy field in the KubeletConfiguration struct. A Secret can be either propagated by watch (default), ttl-based, or simply redirecting all requests directly to the API server. As a result, the total delay from the moment when the Secret is updated to the moment when new keys are projected to the Pod can be as long as the kubelet sync period + cache propagation delay, where the cache propagation delay depends on the chosen cache type (it equals to watch propagation delay, ttl of cache, or zero correspondingly).
			// in KinD cluster, the observatorium-api won't be restarted, it may due to cert-manager webhook or the kubelet sync period + cache propagation delay
			// so manually delete the pod to force it restart
			for _, hubPodName := range hubPodsName {
				utils.DeletePod(testOptions, true, MCO_NAMESPACE, hubPodName)
			}
		}

		By(fmt.Sprintf("Waiting for old pods removed: %v and new pods created", hubPodsName))
		Eventually(func() bool {
			err1, appPodList := utils.GetPodList(testOptions, true, MCO_NAMESPACE, "app.kubernetes.io/name=observatorium-api")
			err2, rbacPodList := utils.GetPodList(testOptions, true, MCO_NAMESPACE, "app=rbac-query-proxy")
			if err1 == nil && err2 == nil {
				if len(hubPodsName) != len(appPodList.Items)+len(rbacPodList.Items) {
					klog.V(1).Infof("Wrong number of pods: <%d> observatorium-api pods and <%d> rbac-query-proxy pods", len(appPodList.Items), len(rbacPodList.Items))
					return false
				}
				for _, oldPodName := range hubPodsName {
					for _, pod := range appPodList.Items {
						if oldPodName == pod.Name {
							klog.V(1).Infof("<%s> not removed yet", oldPodName)
							return false
						}
					}
					for _, pod := range rbacPodList.Items {
						if oldPodName == pod.Name {
							klog.V(1).Infof("<%s> not removed yet", oldPodName)
							return false
						}
					}
				}
				return true
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By(fmt.Sprintf("Waiting for old pod removed: %s and new pod created", collectorPodName))
		Eventually(func() bool {
			err, podList := utils.GetPodList(testOptions, false, MCO_ADDON_NAMESPACE, "component=metrics-collector")
			if err == nil {
				for _, pod := range podList.Items {
					if pod.Name != collectorPodName {
						return true
					}
				}
			} else {
				return false
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
