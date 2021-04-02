// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package utils

import (
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
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
	OCM_ADDON_GROUP               = "addon.open-cluster-management.io"
)

func NewMCOGVRV1BETA1() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    MCO_GROUP,
		Version:  "v1beta1",
		Resource: "multiclusterobservabilities"}
}

func NewMCOGVRV1BETA2() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    MCO_GROUP,
		Version:  "v1beta2",
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

func NewMCOClusterManagementAddonsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    OCM_ADDON_GROUP,
		Version:  "v1alpha1",
		Resource: "clustermanagementaddons"}
}

func NewMCOManagedClusterAddonsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    OCM_ADDON_GROUP,
		Version:  "v1alpha1",
		Resource: "managedclusteraddons"}
}

func NewMCOMObservatoriumGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "core.observatorium.io",
		Version:  "v1alpha1",
		Resource: "observatoria"}
}

func ModifyMCOAvailabilityConfig(opt TestOptions, availabilityConfig string) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	spec := mco.Object["spec"].(map[string]interface{})
	spec["availabilityConfig"] = availabilityConfig
	_, updateErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Update(mco, metav1.UpdateOptions{})
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

	mcoPods, err := hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
	if err != nil {
		return []corev1.Pod{}, err
	}
	return mcoPods.Items, nil
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
				isReady = true
				break
			}
		}

		// only print not ready pod status
		if !isReady {
			klog.V(1).Infof("Pod <%s> is not <Ready> on <%s> status due to %#v\n", pod.Name, pod.Status.Phase, pod.Status)
		}
	}
}

func PrintMCOObject(opt TestOptions) {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		klog.V(1).Infof("Failed to get mco object")
		return
	}
	klog.V(1).Infof("MCO spec: %+v\n", mco.Object["spec"])
	klog.V(1).Infof("MCO status: %+v\n", mco.Object["status"])
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
				isReady = true
				break
			}
		}
		// only print not ready pod status
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
	ignorePods := MCO_CR_NAME + "-thanos-store-shard-1-0," + MCO_CR_NAME + "-thanos-store-shard-2-0"

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

		if pod.Spec.Affinity == nil {
			return fmt.Errorf("Failed to ckeck affinity for pod: %v" + pod.GetName())
		}

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
			err = fmt.Errorf("Deployment %s should have 1 but got %d ready replicas",
				deploymentName,
				deployment.Status.ReadyReplicas)
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
		MCO_CR_NAME + "-grafana",
		MCO_CR_NAME + "-observatorium-api",
		MCO_CR_NAME + "-thanos-query",
		MCO_CR_NAME + "-thanos-query-frontend",
		MCO_CR_NAME + "-thanos-receive-controller",
		MCO_CR_NAME + "-observatorium-operator",
		MCO_CR_NAME + "-rbac-query-proxy",
	}

	for _, deploymentName := range expectedDeploymentNames {
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}

		if deployment.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Deployment %s should have 1 but got %d ready replicas",
				deploymentName,
				deployment.Status.ReadyReplicas)
			return err
		}
	}

	statefulsets := client.AppsV1().StatefulSets(MCO_NAMESPACE)
	expectedStatefulSetNames := []string{
		MCO_CR_NAME + "-alertmanager",
		MCO_CR_NAME + "-thanos-compact",
		MCO_CR_NAME + "-thanos-receive-default",
		MCO_CR_NAME + "-thanos-rule",
		MCO_CR_NAME + "-thanos-store-memcached",
		MCO_CR_NAME + "-thanos-store-shard-0",
	}

	for _, statefulsetName := range expectedStatefulSetNames {
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}

		if statefulset.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Statefulset %s should have 1 but got %d ready replicas",
				statefulsetName,
				statefulset.Status.ReadyReplicas)
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
		MCO_CR_NAME + "-grafana",
		MCO_CR_NAME + "-observatorium-api",
		MCO_CR_NAME + "-thanos-query",
		MCO_CR_NAME + "-thanos-query-frontend",
		MCO_CR_NAME + "-rbac-query-proxy",
	}

	for _, deploymentName := range expectedDeploymentNames {
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}

		if deployment.Status.ReadyReplicas != 2 {
			err = fmt.Errorf("Deployment %s should have 2 but got %d ready replicas",
				deploymentName,
				deployment.Status.ReadyReplicas)
			return err
		}
	}

	statefulsets := client.AppsV1().StatefulSets(MCO_NAMESPACE)
	expectedStatefulSetNames := []string{
		MCO_CR_NAME + "-alertmanager",
		MCO_CR_NAME + "-thanos-receive-default",
		MCO_CR_NAME + "-thanos-rule",
		MCO_CR_NAME + "-thanos-store-memcached",
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
			err = fmt.Errorf("Statefulset %s should have 3 but got %d ready replicas",
				statefulsetName,
				statefulset.Status.ReadyReplicas)
			return err
		}
	}

	expectedStatefulSetNames = []string{
		MCO_CR_NAME + "-thanos-compact",
		MCO_CR_NAME + "-thanos-store-shard-0",
		MCO_CR_NAME + "-thanos-store-shard-1",
		MCO_CR_NAME + "-thanos-store-shard-2",
	}

	for _, statefulsetName := range expectedStatefulSetNames {
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}

		if statefulset.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Statefulset %s should have 1 but got %d ready replicas",
				statefulsetName,
				statefulset.Status.ReadyReplicas)
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
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	spec := mco.Object["spec"].(map[string]interface{})
	spec["nodeSelector"] = map[string]string{"kubernetes.io/os": "linux"}
	retentionConfig := spec["retentionConfig"].(map[string]interface{})
	retentionConfig["retentionResolutionRaw"] = "3d"

	_, updateErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Update(mco, metav1.UpdateOptions{})
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
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}
	spec := mco.Object["spec"].(map[string]interface{})
	retentionConfig := spec["retentionConfig"].(map[string]interface{})
	spec["nodeSelector"] = map[string]string{}
	retentionConfig["retentionResolutionRaw"] = "5d"

	_, updateErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Update(mco, metav1.UpdateOptions{})
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
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	spec := mco.Object["spec"].(map[string]interface{})
	spec["retentionResolutionRaw"] = "3d"
	_, updateErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Update(mco, metav1.UpdateOptions{})
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
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
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
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	observabilityAddonSpec := mco.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})
	observabilityAddonSpec["enableMetrics"] = enable
	_, updateErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Update(mco, metav1.UpdateOptions{})
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
	mco, getErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	observabilityAddonSpec := mco.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})
	observabilityAddonSpec["interval"] = interval
	_, updateErr := clientDynamic.Resource(NewMCOGVRV1BETA2()).Update(mco, metav1.UpdateOptions{})
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
	return clientDynamic.Resource(NewMCOGVRV1BETA2()).Delete(MCO_CR_NAME, &metav1.DeleteOptions{})
}

func CheckMCOConversion(opt TestOptions, v1beta1tov1beta2GoldenPath string) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	getMCO, err := clientDynamic.Resource(NewMCOGVRV1BETA2()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if err != nil {
		return err
	}

	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	yamlB, err := ioutil.ReadFile(v1beta1tov1beta2GoldenPath)
	if err != nil {
		return err
	}

	expectedMCO := &unstructured.Unstructured{}
	_, _, err = decUnstructured.Decode(yamlB, nil, expectedMCO)
	if err != nil {
		return err
	}

	getMCOSpec := getMCO.Object["spec"].(map[string]interface{})
	expectedMCOSpec := expectedMCO.Object["spec"].(map[string]interface{})

	for k, v := range expectedMCOSpec {
		val, ok := getMCOSpec[k]
		if !ok {
			return fmt.Errorf("%s not found in ", k)
		}
		if !reflect.DeepEqual(val, v) {
			return fmt.Errorf("%+v and %+v are not equal!", val, v)
		}
	}
	return nil
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
