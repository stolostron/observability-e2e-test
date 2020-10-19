package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	MCO_OPERATOR_NAMESPACE = "open-cluster-management"
	MCO_NAMESPACE          = "open-cluster-management-observability"
	MCO_LABEL              = "name=multicluster-observability-operator"
	MCO_PULL_SECRET_NAME   = "multiclusterhub-operator-pull-secret"
	OBJ_SECRET_NAME        = "thanos-object-storage"
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
      key: thanos.yaml`,
		name)

	return []byte(instance)
}

func DeleteMCOInstance(url string, kubeconfig string, context string) error {
	instanceName := "observability"
	yamlByte := NewMCOInstanceYaml(instanceName)
	obj := &unstructured.Unstructured{}
	err := yaml.Unmarshal([]byte(yamlByte), obj)
	if err != nil {
		return err
	}
	var group string
	var version string
	if v, ok := obj.Object["apiVersion"]; !ok {
		return fmt.Errorf("apiVersion attribute not found in %s", yamlByte)
	} else {
		apiVersionArray := strings.Split(v.(string), "/")
		if len(apiVersionArray) != 2 {
			return fmt.Errorf("apiVersion malformed in %s", yamlByte)
		}
		group = apiVersionArray[0]
		version = apiVersionArray[1]
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: "multiclusterobservabilities"}
	clientDynamic := NewKubeClientDynamic(url, kubeconfig, context)
	return clientDynamic.Resource(gvr).Delete(instanceName, &metav1.DeleteOptions{})
}

func CreatePullSecret(url string, kubeconfig string, context string) error {
	clientKube := NewKubeClient(url, kubeconfig, context)
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

	_, err := clientKube.CoreV1().Secrets(pullSecret.Namespace).Create(pullSecret)
	return err
}

func CreateMCONamespace(url string, kubeconfig string, context string) error {
	ns := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`,
		MCO_NAMESPACE)

	return Apply(url, kubeconfig, context, []byte(ns))
}

func CreateObjSecret(url string, kubeconfig string, context string) error {

	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		return errors.New("failed to get s3 BUCKET env")
	}

	region := os.Getenv("REGION")
	if region == "" {
		return errors.New("failed to get s3 REGION env")
	}

	accessKey := os.Getenv("ACCESSKEY")
	if accessKey == "" {
		return errors.New("failed to get aws ACCESSKEY env")
	}

	secretKey := os.Getenv("SECRETKEY")
	if secretKey == "" {
		return errors.New("failed to get aws SECRETKEY env")
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

	return Apply(url, kubeconfig, context, []byte(objSecret))
}

func UninstallMCO(url string, kubeconfig string, context string) error {
	deleteMCOErr := DeleteMCOInstance(url, kubeconfig, context)
	if deleteMCOErr != nil {
		return deleteMCOErr
	}

	clientKube := NewKubeClient(url, kubeconfig, context)
	deletePullSecretErr := clientKube.CoreV1().Secrets(MCO_NAMESPACE).Delete(MCO_PULL_SECRET_NAME, &metav1.DeleteOptions{})
	if deletePullSecretErr != nil {
		return deletePullSecretErr
	}

	deleteObjSecretErr := clientKube.CoreV1().Secrets(MCO_NAMESPACE).Delete(OBJ_SECRET_NAME, &metav1.DeleteOptions{})
	if deleteObjSecretErr != nil {
		return deleteObjSecretErr
	}

	return nil
}
