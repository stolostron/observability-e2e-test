package utils

import (
	"errors"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
)

const (
	MCO_NAMESPACE        = "open-cluster-management-observability"
	MCO_CR_NAME          = "observability"
	MCO_LABEL            = "name=multicluster-observability-operator"
	MCO_PULL_SECRET_NAME = "multiclusterhub-operator-pull-secret"
	OBJ_SECRET_NAME      = "thanos-object-storage"
	MCO_GROUP            = "observability.open-cluster-management.io"
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

func ModifyMCOAvailabilityConfig(opt TestOptions) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	spec := mco.Object["spec"].(map[string]interface{})
	spec["availabilityConfig"] = "Basic"
	_, updateErr := clientDynamic.Resource(NewMCOGVR()).Update(mco, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}
	return nil
}

func PrintAllMCOPodsStatus(opt TestOptions) {
	hubClient := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	mcoPods, err := hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Failed to get all MCO pods")
	}

	for _, pod := range mcoPods.Items {
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

func CheckMCOComponentsReady(opt TestOptions) error {
	deployments := []string{
		"grafana",
		MCO_CR_NAME + "-observatorium-observatorium-api",
		MCO_CR_NAME + "-observatorium-thanos-query",
		MCO_CR_NAME + "-observatorium-thanos-query-frontend",
		MCO_CR_NAME + "-observatorium-thanos-receive-controller",
		"observatorium-operator",
		"rbac-query-proxy",
	}

	deploymentsErr := HaveDeploymentsInNamespace(
		opt.HubCluster,
		opt.KubeConfig,
		MCO_NAMESPACE,
		deployments)

	statefulsets := []string{
		"alertmanager",
		MCO_CR_NAME + "-observatorium-thanos-compact",
		MCO_CR_NAME + "-observatorium-thanos-receive-default",
		MCO_CR_NAME + "-observatorium-thanos-rule",
		MCO_CR_NAME + "-observatorium-thanos-store-memcached",
		MCO_CR_NAME + "-observatorium-thanos-store-shard-0",
		MCO_CR_NAME + "-observatorium-thanos-store-shard-1",
		MCO_CR_NAME + "-observatorium-thanos-store-shard-2",
	}

	statefulsetsErr := HaveStatefulSetsInNamespace(
		opt.HubCluster,
		opt.KubeConfig,
		MCO_NAMESPACE,
		statefulsets)

	if deploymentsErr != nil || statefulsetsErr != nil {
		PrintAllMCOPodsStatus(opt)
		return errors.New("Failed to check MCO components ready")
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
		MCO_CR_NAME + "-observatorium-observatorium-api",
		MCO_CR_NAME + "-observatorium-thanos-query",
		MCO_CR_NAME + "-observatorium-thanos-query-frontend",
		MCO_CR_NAME + "-observatorium-thanos-receive-controller",
		"observatorium-operator",
		"rbac-query-proxy",
	}

	for _, deploymentName := range expectedDeploymentNames {
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}

		if deployment.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Expect 1 but got %d ready replicas", deployment.Status.ReadyReplicas)
			klog.Errorln(err)
			return err
		}
	}

	statefulsets := client.AppsV1().StatefulSets(MCO_NAMESPACE)
	expectedStatefulSetNames := []string{
		"alertmanager",
		MCO_CR_NAME + "-observatorium-thanos-compact",
		MCO_CR_NAME + "-observatorium-thanos-receive-default",
		MCO_CR_NAME + "-observatorium-thanos-rule",
		MCO_CR_NAME + "-observatorium-thanos-store-memcached",
		MCO_CR_NAME + "-observatorium-thanos-store-shard-0",
	}

	for _, statefulsetName := range expectedStatefulSetNames {
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}

		if statefulset.Status.ReadyReplicas != 1 {
			err = fmt.Errorf("Expect 1 but got %d ready replicas", statefulset.Status.ReadyReplicas)
			klog.Errorln(err)
			return err
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

func ModifyMCOobservabilityAddonSpec(opt TestOptions) error {
	clientDynamic := NewKubeClientDynamic(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	mco, getErr := clientDynamic.Resource(NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	observabilityAddonSpec := mco.Object["spec"].(map[string]interface{})["observabilityAddonSpec"].(map[string]interface{})
	observabilityAddonSpec["enableMetrics"] = false
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
	return clientDynamic.Resource(NewMCOGVR()).Delete(MCO_CR_NAME, &metav1.DeleteOptions{})
}

func CreatePullSecret(opt TestOptions, mcoNs string) error {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	name := "multiclusterhub-operator-pull-secret"
	pullSecret, errGet := clientKube.CoreV1().Secrets(mcoNs).Get(name, metav1.GetOptions{})
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
		return errors.New("failed to get s3 BUCKET env")
	}

	region := os.Getenv("REGION")
	if region == "" {
		return errors.New("failed to get s3 REGION env")
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKey == "" {
		return errors.New("failed to get aws AWS_ACCESS_KEY_ID env")
	}

	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretKey == "" {
		return errors.New("failed to get aws AWS_SECRET_ACCESS_KEY env")
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
