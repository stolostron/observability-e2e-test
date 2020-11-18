package main_test

import (
	"fmt"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

func NewThanosConfigMap(name, namespace string) *corev1.ConfigMap {
	instance := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: hello%s
  namespace: open-cluster-management-observability%s
data:
  custom_rules.yaml: |
    groups:
      - name: cluster-health
        rules:
        - alert: ClusterCPUHealth
          annotations:
            summary: Fires when CPU Utilization on a Cluster gets too high.
            description: "The Cluster has a high CPU usage: {{ $value }} core for {{ $labels.cluster }} {{ $labels.clusterID }}."  
          expr: |
            max(cluster:cpu_usage_cores:sum) by (clusterID, cluster) > 0
          for: 3m
          labels:
            cluster: "{{ $labels.cluster }}"
            clusterID: "{{ $labels.clusterID }}"
            severity: critical`,
		name,
		namespace,
	)

	obj := &corev1.ConfigMap{}
	err := yaml.Unmarshal([]byte(instance), obj)

	// TODO: fix name and namespace addition to configmap.
	obj.Name = name
	obj.Namespace = namespace

	if err != nil {
		klog.V(3).Infof("%v", err)
		return nil
	}

	return obj
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

	It("should have the expected stateful sets (alert/g0)", func() {
		By("Checking if STS: Alertmanager and observability-observatorium-thanos-rule is existed")
		expectedStatefulSet := [...]string{"alertmanager", "observability-observatorium-thanos-rule"}

		for _, resource := range expectedStatefulSet {
			sts, _ := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(resource, metav1.GetOptions{})

			By("Having MCO installed, the statefulset: " + sts.GetName() + " should have 3 replicas")
			Expect(sts.Status.Replicas).To(Equal(int32(3)))

			if sts.GetName() == "alertmanager" {
				By("The statefulset: " + sts.GetName() + " should have the appropriate secret mounted")
				Expect(len(sts.Spec.Template.Spec.Volumes)).Should(BeNumerically(">", 0))
				Expect(sts.Spec.Template.Spec.Volumes[0].Secret.SecretName).To(Equal("alertmanager-config"))
			}

			if sts.GetName() == "observability-observatorium-thanos-rule" {
				By("The statefulset: " + sts.GetName() + " should have the appropriate configmap mounted")
				Expect(len(sts.Spec.Template.Spec.Volumes)).Should(BeNumerically(">", 0))
				Expect(sts.Spec.Template.Spec.Volumes[0].ConfigMap.Name).To(Equal("thanos-ruler-default-rules"))
			}
		}
	})

	expectedConfigMap := [...]string{"thanos-ruler-default-rules", "thanos-ruler-custom-rules"}

	It("should have the expected configmap (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-default-rules is existed")
		cm, _ := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[0], metav1.GetOptions{})
		Expect(cm.ResourceVersion).ShouldNot(BeEmpty())
	})

	It("should not have the CM: thanos-ruler-custom-rules (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-custom-rules not existed")
		cm, _ := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[1], metav1.GetOptions{})
		Expect(cm.ResourceVersion).Should(BeEmpty())
		klog.V(3).Infof("Configmap %s does not exist", expectedConfigMap[1])
	})

	It("should create the CM: thanos-ruler-custom-rules (alert/g0)", func() {
		By("Creating CM: " + expectedConfigMap[1])
		obj := NewThanosConfigMap(expectedConfigMap[1], MCO_NAMESPACE)

		klog.V(3).Infof("Creating Configmap %s", expectedConfigMap[1])
		cm, _ := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Create(obj)

		Expect(cm.Name).To(Equal("thanos-ruler-custom-rules"))
		Expect(cm.Data).ShouldNot(BeEmpty())

		By("Checking to see if Configmap was actually created")
		cm, getErr := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[1], metav1.GetOptions{})

		if getErr != nil {
			klog.V(3).Infof("Error getting configmap: %v", getErr)
		} else {
			klog.V(3).Info("Successfully got configmap...")
		}
	})

	It("should expect custom config to be deleted (alert/g0)", func() {
		deleteConfigMapErr := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(expectedConfigMap[1], &metav1.DeleteOptions{})
		Expect(deleteConfigMapErr).Should(BeNil())

		_, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[1], metav1.GetOptions{})
		Expect(err).ToNot(BeNil())
	})

	It("should have the expected secret (alert/g0)", func() {
		By("Checking if SECRETS: alertmanager-config is existed")
		secretList, _ := hubClient.CoreV1().Secrets(MCO_NAMESPACE).List(metav1.ListOptions{})
		expectedSecret := [...]string{"alertmanager-config"}

		for _, resource := range expectedSecret {
			for _, secret := range secretList.Items {
				if resource == secret.GetName() {
					klog.V(3).Infof("Found Secret: %s", secret.GetName())
				}
			}
		}
	})
})
