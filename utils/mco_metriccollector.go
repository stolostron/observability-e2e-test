package utils

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func GetMetricsCollectorPodList(opt TestOptions) (error, *v1.PodList) {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	if len(opt.ManagedClusters) > 0 {
		clientKube = NewKubeClient(
			opt.ManagedClusters[0].MasterURL,
			opt.ManagedClusters[0].KubeConfig,
			opt.ManagedClusters[0].KubeContext)
	}
	listOption := metav1.ListOptions{
		LabelSelector: "component=metrics-collector",
	}
	podList, err := clientKube.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(listOption)
	if err != nil {
		klog.Errorf("Failed to get metrics collector pod list due to %v", err)
	}
	return err, podList
}
