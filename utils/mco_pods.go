package utils

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func GetPodList(opt TestOptions, isHub bool, labelSelector string) (error, *v1.PodList) {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	if !isHub && len(opt.ManagedClusters) > 0 {
		clientKube = NewKubeClient(
			opt.ManagedClusters[0].MasterURL,
			opt.ManagedClusters[0].KubeConfig,
			opt.ManagedClusters[0].KubeContext)
	}
	listOption := metav1.ListOptions{}
	if labelSelector != "" {
		listOption.LabelSelector = labelSelector
	}
	podList, err := clientKube.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(listOption)
	if err != nil {
		klog.Errorf("Failed to get pod list using labelselector %s due to %v", labelSelector, err)
	}
	if podList != nil && len(podList.Items) == 0 {
		klog.V(1).Infof("No pod found for labelselector %s", labelSelector)
	}
	return err, podList
}
