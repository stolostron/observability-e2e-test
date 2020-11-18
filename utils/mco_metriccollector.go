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

	listOption := metav1.ListOptions{
		LabelSelector: "component=metrics-collector",
	}
	podList, err := clientKube.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(listOption)
	if err != nil {
		klog.V(1).Infof("Failed to get metrics collector pod list due to %v", err)
	}
	for _, pod := range podList.Items {
		klog.V(1).Infof("pod name is " + pod.Name)
	}
	return err, podList
}
