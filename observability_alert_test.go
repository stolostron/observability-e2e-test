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

func ModifyAlertManagerSecret(obj *corev1.Secret) map[string][]byte {
	instance := fmt.Sprintf(`global:
  resolve_timeout: 5m
route:
  receiver: default-receiver
  routes:
    - match:
        alertname: Watchdog 
      receiver: default-receiver
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 30m
receivers:
  - name: default-receiver
    slack_configs:
	- api_url: https://hooks.slack.com/services/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
      send_resolved: true
      channel: team-observability-test
      footer: |
        {{ .CommonLabels.cluster }}
      mrkdwn_in:
        - text
      title: '[{{ .Status | toUpper }}] {{ .CommonLabels.alertname }} ({{ .CommonLabels.severity }})'
      text: |- 
        {{ range .Alerts }}
          *Alerts:* {{ .Annotations.summary }}
          *Description:* {{ .Annotations.description }}
          *Details:*
          {{ range .Labels.SortedPairs }} • *{{ .Name }}:* {{ .Value }}
          {{ end }}
        {{ end }}
      title_link: https://multicloud-console.apps.%s/grafana/explore?orgId=1&left=["now-1h","now","Observatorium",{"expr":"ALERTS{alertname=\"{{ .CommonLabels.alertname }}\"}","context":"explore"},{"mode":"Metrics"},{"ui":[true,true,true,"none"]}]
`, testOptions.HubCluster.BaseDomain)

	data := make(map[string][]byte)
	data["alertmanager.yaml"] = []byte(instance)
	return data
}

func RevertBackToDefaultAlertManagerConfig() map[string][]byte {
	instance := fmt.Sprintf(`global:
  resolve_timeout: 5m
receivers:
  - name: "null"
route:
  receiver: "null"
  routes:
    - match:
        alertname: Watchdog
      receiver: "null"
  "group_by": ['namespace']
  "group_interval": "5m"
  "group_wait": "30s"
  "repeat_interval": "12h"
`)

	data := make(map[string][]byte)
	data["alertmanager.yaml"] = []byte(instance)

	return data
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

	expectedStatefulSet := [...]string{"alertmanager", "observability-observatorium-thanos-rule"}
	expectedConfigMap := [...]string{"thanos-ruler-default-rules", "thanos-ruler-custom-rules"}
	expectedSecret := "alertmanager-config"

	It("should have the expected stateful sets (alert/g0)", func() {
		By("Checking if STS: Alertmanager and observability-observatorium-thanos-rule is existed")

		for _, resource := range expectedStatefulSet {
			sts, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(resource, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

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

	It("should have the expected configmap (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-default-rules is existed")
		cm, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[0], metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.ResourceVersion).ShouldNot(BeEmpty())
	})

	It("should not have the CM: thanos-ruler-custom-rules (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-custom-rules not existed")
		cm, _ := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[1], metav1.GetOptions{})
		Expect(cm.ResourceVersion).Should(BeEmpty())
		klog.V(3).Infof("Configmap %s does not exist", expectedConfigMap[1])
	})

	It("should create the CM: thanos-ruler-custom-rules (alert/g0)", func() {
		By("Creating CM: thanos-ruler-custom-rules")
		obj := NewThanosConfigMap(expectedConfigMap[1], MCO_NAMESPACE)

		klog.V(3).Infof("Creating Configmap %s", expectedConfigMap[1])
		cm, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Create(obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(cm.Name).To(Equal("thanos-ruler-custom-rules"))
		Expect(cm.Data).ShouldNot(BeEmpty())

		By("Checking to see if Configmap was actually created")
		cm, err = hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[1], metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		klog.V(3).Info("Successfully got configmap...")
	})

	It("should expect custom config to be deleted (alert/g0)", func() {
		By("Deleting the CM: thanos-ruler-custom-rules")
		err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(expectedConfigMap[1], &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Trying to get the CM: thanos-ruler-custom-rules we can verify if the configmap is indeed deleted")
		_, err = hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(expectedConfigMap[1], metav1.GetOptions{})
		Expect(err).To(HaveOccurred())
	})

	It("should have the expected secret (alert/g0)", func() {
		By("Checking if SECRETS: alertmanager-config is existed")
		secret, err := hubClient.CoreV1().Secrets(MCO_NAMESPACE).Get(expectedSecret, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(secret.GetName()).To(Equal("alertmanager-config"))
		klog.V(3).Infof("Successfully got secret... %s", secret.GetName())
	})

	It("should modify the SECRET: alertmanager-config (alert/g0)", func() {
		By("Editing the secret, we should be able to add the third partying tools integrations")
		scrt, err := hubClient.CoreV1().Secrets(MCO_NAMESPACE).Get(expectedSecret, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		// We need to store the older version, so we can revert back to that data.
		obj := &corev1.Secret{}
		obj.ObjectMeta = scrt.ObjectMeta
		obj.Data = ModifyAlertManagerSecret(obj)

		_, err = hubClient.CoreV1().Secrets(MCO_NAMESPACE).Update(obj)
		Expect(err).NotTo(HaveOccurred())
	})
})
