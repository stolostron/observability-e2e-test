package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	secretName   = "observability-server-certs"
	caSecretName = "observability-server-ca-certs"
)

func DeleteCertSecret(opt TestOptions) error {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	klog.V(1).Infof("Delete certificate secret")
	err := clientKube.CoreV1().Secrets(MCO_NAMESPACE).Delete(caSecretName, &metav1.DeleteOptions{})
	if err != nil {
		klog.V(1).Infof("Failed to delete certificate secret %s due to %v", caSecretName, err)
		return err
	}
	err = clientKube.CoreV1().Secrets(MCO_NAMESPACE).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil {
		klog.V(1).Infof("Failed to delete certificate secret %s due to %v", secretName, err)
		return err
	}
	return err
}
