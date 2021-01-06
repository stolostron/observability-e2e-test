package utils

import (
	b64 "encoding/base64"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
)

const (
	MCO_OPERATOR_NAMESPACE        = "open-cluster-management"
	MCO_CR_NAME                   = "observability"
	MCO_COMPONENT_LABEL           = "observability.open-cluster-management.io/name=" + MCO_CR_NAME
	OBSERVATORIUM_COMPONENT_LABEL = "app.kubernetes.io/part-of=observatorium"
	MCO_NAMESPACE                 = "open-cluster-management-observability"
	MCO_ADDON_NAMESPACE           = "open-cluster-management-addon-observability"
	MCO_PULL_SECRET_NAME          = "multiclusterhub-operator-pull-secret"
	OBJ_SECRET_NAME               = "thanos-object-storage"
	MCO_GROUP                     = "observability.open-cluster-management.io"
	OCM_WORK_GROUP                = "work.open-cluster-management.io"
	OCM_CLUSTER_GROUP             = "cluster.open-cluster-management.io"
)

func NewMCOInstanceYaml(name string) []byte {
	instance := fmt.Sprintf(`apiVersion: observability.open-cluster-management.io/v1beta1
kind: MultiClusterObservability
metadata:
  name: %s
spec:
  storageConfigObject:
    metricObjectStorage:
      name: thanos-object-storage
	  key: thanos.yaml
	statefulSetSize: 4Gi`,
		name)

	return []byte(instance)
}

func NewMCOGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    MCO_GROUP,
		Version:  "v1beta1",
		Resource: "multiclusterobservabilities"}
}

func NewMCOAddonGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    MCO_GROUP,
		Version:  "v1beta1",
		Resource: "observabilityaddons"}
}

func NewOCMManifestworksGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    OCM_WORK_GROUP,
		Version:  "v1",
		Resource: "manifestworks"}
}

func NewOCMManagedClustersGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    OCM_CLUSTER_GROUP,
		Version:  "v1",
		Resource: "managedclusters"}
}

func ModifyMCOAvailabilityConfig(opt TestOptions, availabilityConfig string) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	spec := mco.Object["spec"].(map[string]interface{})
	spec["availabilityConfig"] = availabilityConfig
	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func ModifyMCONodeSelector(opt TestOptions, nodeSelector map[string]string) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	spec := mco.Object["spec"].(map[string]interface{})
	spec["nodeSelector"] = nodeSelector
	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func GetAllMCOPods(opt TestOptions) ([]corev1.Pod, error) {
	hubClient := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	mcoOpt := metav1.ListOptions{LabelSelector: MCO_COMPONENT_LABEL}
	mcoPods, err := hubClient.CoreV1().Pods(MCO_NAMESPACE).List(mcoOpt)
	if err != nil {
		return []corev1.Pod{}, err
	}

	obsOpt := metav1.ListOptions{LabelSelector: OBSERVATORIUM_COMPONENT_LABEL}
	obsPods, err := hubClient.CoreV1().Pods(MCO_NAMESPACE).List(obsOpt)
	if err != nil {
		return []corev1.Pod{}, err
	}

	return append(mcoPods.Items, obsPods.Items...), nil
}

func PrintAllMCOPodsStatus(opt TestOptions) {
	podList, err := GetAllMCOPods(opt)
	if err != nil {
		klog.Errorf("Failed to get all MCO pods")
	}

	for _, pod := range podList {
		isReady := false
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" {
				klog.V(1).Infof("Pod <%s> is <Ready> on <%s> status\n", pod.Name, pod.Status.Phase)
				isReady = true
				break
			}
		}

		if !isReady {
			klog.V(1).Infof("Pod <%s> is not <Ready> on <%s> status due to %#v\n", pod.Name, pod.Status.Phase, pod.Status)
		}
	}
}

func GetAllOBAPods(opt TestOptions) ([]corev1.Pod, error) {
	clientKube := getKubeClient(opt, false)

	obaPods, err := clientKube.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{})
	if err != nil {
		return []corev1.Pod{}, err
	}

	return obaPods.Items, nil
}

func PrintAllOBAPodsStatus(opt TestOptions) {
	podList, err := GetAllOBAPods(opt)
	if err != nil {
		klog.Errorf("Failed to get all OBA pods")
	}

	for _, pod := range podList {
		isReady := false
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" {
				klog.V(1).Infof("Pod <%s> is <Ready> on <%s> status\n", pod.Name, pod.Status.Phase)
				isReady = true
				break
			}
		}

		if !isReady {
			klog.V(1).Infof("Pod <%s> is not <Ready> on <%s> status due to %#v\n", pod.Name, pod.Status.Phase, pod.Status)
		}
	}
}

func CheckAllPodNodeSelector(opt TestOptions) error {
	podList, err := GetAllMCOPods(opt)
	if err != nil {
		return err
	}
	//shard-1-0 and shard-2-0 won't be deleted when switch from High to Basic
	//And cannot apply the nodeSelector to shard-1-0 and shard-2-0
	//https://github.com/open-cluster-management/backlog/issues/6532
	ignorePods := MCO_CR_NAME + "-observatorium-thanos-store-shard-1-0," + MCO_CR_NAME + "-observatorium-thanos-store-shard-2-0"

	for _, pod := range podList {
		if strings.Contains(ignorePods, pod.GetName()) {
			continue
		}

		selecterValue, ok := pod.Spec.NodeSelector["kubernetes.io/os"]
		if !ok || selecterValue != "linux" {
			return fmt.Errorf("Failed to check node selector for pod: %v", pod.GetName())
		}
	}
	return nil
}

func CheckAllPodsAffinity(opt TestOptions) error {
	podList, err := GetAllMCOPods(opt)
	if err != nil {
		return err
	}

	for _, pod := range podList {
		weightedPodAffinityTerms := pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		for _, weightedPodAffinityTerm := range weightedPodAffinityTerms {
			topologyKey := weightedPodAffinityTerm.PodAffinityTerm.TopologyKey
			if (topologyKey == "kubernetes.io/hostname" && weightedPodAffinityTerm.Weight == 30) ||
				(topologyKey == "topology.kubernetes.io/zone" && weightedPodAffinityTerm.Weight == 70) {
			} else {
				return fmt.Errorf("Failed to ckeck affinity for pod: %v" + pod.GetName())
			}
		}
	}
	return nil
}

func CheckOBAComponents(opt TestOptions) error {
	client := getKubeClient(opt, false)
	deployments := client.AppsV1().Deployments(MCO_ADDON_NAMESPACE)
	expectedDeploymentNames := []string{
		"endpoint-observability-operator",
		"metrics-collector-deployment",
	}

	for _, deploymentName := range expectedDeploymentNames {
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}

		if deployment.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Expect 1 but got %d ready replicas", deployment.Status.ReadyReplicas)
			return err
		}
	}

	return nil
}

func CheckMCOComponentsInBaiscMode(opt TestOptions) error {
	client := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	deployments := client.AppsV1().Deployments(MCO_NAMESPACE)
	expectedDeploymentNames := []string{
		"grafana",
		"observability-observatorium-observatorium-api",
		"observability-observatorium-thanos-query",
		"observability-observatorium-thanos-query-frontend",
		"observability-observatorium-thanos-receive-controller",
		"observatorium-operator",
		"rbac-query-proxy",
	}

	for _, deploymentName := range expectedDeploymentNames {
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}

		if deployment.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Expect 1 but got %d ready replicas", deployment.Status.ReadyReplicas)
			return err
		}
	}

	statefulsets := client.AppsV1().StatefulSets(MCO_NAMESPACE)
	expectedStatefulSetNames := []string{
		"alertmanager",
		"observability-observatorium-thanos-compact",
		"observability-observatorium-thanos-receive-default",
		"observability-observatorium-thanos-rule",
		"observability-observatorium-thanos-store-memcached",
		"observability-observatorium-thanos-store-shard-0",
	}

	for _, statefulsetName := range expectedStatefulSetNames {
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}

		if statefulset.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Expect 1 but got %d ready replicas", statefulset.Status.ReadyReplicas)
			return err
		}
	}

	return nil
}

func CheckMCOComponentsInHighMode(opt TestOptions) error {
	client := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	deployments := client.AppsV1().Deployments(MCO_NAMESPACE)
	expectedDeploymentNames := []string{
		"grafana",
		"observability-observatorium-observatorium-api",
		"observability-observatorium-thanos-query",
		"observability-observatorium-thanos-query-frontend",
		"rbac-query-proxy",
	}

	for _, deploymentName := range expectedDeploymentNames {
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}

		if deployment.Status.ReadyReplicas != 2 {
			err = fmt.Errorf("Expect 2 but got %d ready replicas", deployment.Status.ReadyReplicas)
			return err
		}
	}

	statefulsets := client.AppsV1().StatefulSets(MCO_NAMESPACE)
	expectedStatefulSetNames := []string{
		"alertmanager",
		"observability-observatorium-thanos-compact",
		"observability-observatorium-thanos-receive-default",
		"observability-observatorium-thanos-rule",
		"observability-observatorium-thanos-store-memcached",
		// TODO: https://github.com/open-cluster-management/backlog/issues/6532
		// "observability-observatorium-thanos-store-shard-0",
	}

	for _, statefulsetName := range expectedStatefulSetNames {
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}

		if statefulset.Status.ReadyReplicas != 3 {
			err = fmt.Errorf("Expect 3 but got %d ready replicas", statefulset.Status.ReadyReplicas)
			return err
		}
	}

	expectedStatefulSetNames = []string{
		"observability-observatorium-thanos-compact",
		"observability-observatorium-thanos-store-shard-0",
		"observability-observatorium-thanos-store-shard-1",
		"observability-observatorium-thanos-store-shard-2",
	}

	for _, statefulsetName := range expectedStatefulSetNames {
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}

		if statefulset.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Expect 1 but got %d ready replicas", statefulset.Status.ReadyReplicas)
			return err
		}
	}

	return nil
}

// ModifyMCOCR modifies the MCO CR for reconciling. modify multiple parameter to save running time
func ModifyMCOCR(opt TestOptions) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	spec := mco.Object["spec"].(map[string]interface{})
	spec["retentionResolutionRaw"] = "3d"
	spec["nodeSelector"] = map[string]string{"kubernetes.io/os": "linux"}
	spec["availabilityConfig"] = "Basic"

	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

// RevertMCOCRModification revert the previous changes
func RevertMCOCRModification(opt TestOptions) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	spec := mco.Object["spec"].(map[string]interface{})
	spec["retentionResolutionRaw"] = "5d"
	spec["nodeSelector"] = map[string]string{}
	if IsCanaryEnvironment(opt) {
		//KinD cluster does not have enough resource to support High mode
		spec["availabilityConfig"] = "High"
	}

	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func CheckMCOAddon(opt TestOptions) error {
	client := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	if len(opt.ManagedClusters) > 0 {
		client = NewKubeClient(
			opt.ManagedClusters[0].MasterURL,
			opt.ManagedClusters[0].KubeConfig,
			"")
	}
	expectedPodNames := []string{
		"endpoint-observability-operator",
		"metrics-collector-deployment",
	}
	podList, err := client.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	podsn := make(map[string]corev1.PodPhase)
	for _, pod := range podList.Items {
		podsn[pod.Name] = pod.Status.Phase
	}
	for _, podName := range expectedPodNames {
		exist := false
		for key, value := range podsn {
			if strings.HasPrefix(key, podName) && value == "Running" {
				exist = true
			}
		}
		if !exist {
			return fmt.Errorf(podName + " not found")
		}
	}
	return nil
}

func ModifyMCORetentionResolutionRaw(opt TestOptions) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	spec := mco.Object["spec"].(map[string]interface{})
	spec["retentionResolutionRaw"] = "3d"
	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func GetMCOAddonSpecMetrics(opt TestOptions) (bool, error) {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return false, getErr
	}

	enable := mco.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})["enableMetrics"].(bool)
	return enable, nil
}

func ModifyMCOAddonSpecMetrics(opt TestOptions, enable bool) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	observabilityAddonSpec := mco.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})
	observabilityAddonSpec["enableMetrics"] = enable
	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func ModifyMCOAddonSpecInterval(opt TestOptions, interval int64) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	observabilityAddonSpec := mco.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})
	observabilityAddonSpec["interval"] = interval
	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}
func DeleteMCOInstance(opt TestOptions) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	return clientDynamic.Resource(NewMCOGVR()).Delete("observability", &metav1.DeleteOptions{})
}

func CreatePullSecret(opt TestOptions) error {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	namespace := MCO_OPERATOR_NAMESPACE
	name := "multiclusterhub-operator-pull-secret"
	pullSecret, errGet := clientKube.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if errGet != nil {
		return errGet
	}

	pullSecret.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: MCO_NAMESPACE,
	}
	klog.V(1).Infof("Create MCO pull secret")
	_, err := clientKube.CoreV1().Secrets(pullSecret.Namespace).Create(pullSecret)
	return err
}

func CreateMCONamespace(opt TestOptions) error {
	ns := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`,
		MCO_NAMESPACE)
	klog.V(1).Infof("Create MCO namespaces")
	return Apply(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext,
		[]byte(ns))
}

func CreateObjSecret(opt TestOptions) error {

	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		return fmt.Errorf("failed to get s3 BUCKET env")
	}

	region := os.Getenv("REGION")
	if region == "" {
		return fmt.Errorf("failed to get s3 REGION env")
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKey == "" {
		return fmt.Errorf("failed to get aws AWS_ACCESS_KEY_ID env")
	}

	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretKey == "" {
		return fmt.Errorf("failed to get aws AWS_SECRET_ACCESS_KEY env")
	}

	objSecret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
stringData:
  thanos.yaml: |
    type: s3
    config:
      bucket: %s
      endpoint: s3.%s.amazonaws.com
      insecure: false
      access_key: %s
      secret_key: %s
type: Opaque`,
		OBJ_SECRET_NAME,
		MCO_NAMESPACE,
		bucket,
		region,
		accessKey,
		secretKey)
	klog.V(1).Infof("Create MCO object storage secret")
	return Apply(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext,
		[]byte(objSecret))
}

func UninstallMCO(opt TestOptions) error {
	klog.V(1).Infof("Delete MCO instance")
	deleteMCOErr := DeleteMCOInstance(opt)
	if deleteMCOErr != nil {
		return deleteMCOErr
	}

	klog.V(1).Infof("Delete MCO pull secret")
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	deletePullSecretErr := clientKube.CoreV1().Secrets(MCO_NAMESPACE).Delete(MCO_PULL_SECRET_NAME, &metav1.DeleteOptions{})
	if deletePullSecretErr != nil {
		return deletePullSecretErr
	}

	klog.V(1).Infof("Delete MCO object storage secret")
	deleteObjSecretErr := clientKube.CoreV1().Secrets(MCO_NAMESPACE).Delete(OBJ_SECRET_NAME, &metav1.DeleteOptions{})
	if deleteObjSecretErr != nil {
		return deleteObjSecretErr
	}

	return nil
}

func CreateCustomAlertConfigYaml(baseDomain string) []byte {
	global := fmt.Sprintf(`global:
  resolve_timeout: 5m
route:
  receiver: default-receiver
  routes:
    - match:
        alertname: Watchdog
      receiver: default-receiver
  group_by: ['alertname', 'cluster']
  group_wait: 5s
  group_interval: 5s
  repeat_interval: 2m
receivers:
  - name: default-receiver
    slack_configs:
    - api_url: https://hooks.slack.com/services/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
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
`, baseDomain)
	encodedGlobal := b64.StdEncoding.EncodeToString([]byte(global))

	instance := fmt.Sprintf(`kind: Secret
apiVersion: v1
metadata:
  name: alertmanager-config
  namespace: open-cluster-management-observability
data:
  alertmanager.yaml: >-
    %s
`, encodedGlobal)

	return []byte(instance)
}
