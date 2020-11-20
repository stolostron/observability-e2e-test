package utils

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	whitelistCMname = "observability-metrics-custom-whitelist"
)

func CreateMetricsWhitelist(opt TestOptions) error {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	metricsList := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      whitelistCMname,
			Namespace: MCO_NAMESPACE,
		},
		Data: map[string]string{"metrics_list.yaml": `
  names:
    - node_memory_Active_bytes
`},
	}
	klog.V(1).Infof("Create metrics whitelist configmap")
	_, err := clientKube.CoreV1().ConfigMaps(MCO_NAMESPACE).Create(metricsList)
	return err
}

func DeleteMetricsWhitelist(opt TestOptions) error {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)

	klog.V(1).Infof("Delete metrics whitelist configmap")
	err := clientKube.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(whitelistCMname, &metav1.DeleteOptions{})
	return err
}
