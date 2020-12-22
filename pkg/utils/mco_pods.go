package utils

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func GetPodList(opt TestOptions, isHub bool, namespace string, labelSelector string) (error, *v1.PodList) {
	clientKube := getKubeClient(opt, isHub)
	listOption := metav1.ListOptions{}
	if labelSelector != "" {
		listOption.LabelSelector = labelSelector
	}
	podList, err := clientKube.CoreV1().Pods(namespace).List(listOption)
	if err != nil {
		klog.Errorf("Failed to get pod list in namespace %s using labelselector %s due to %v", namespace, labelSelector, err)
		return err, podList
	}
	if podList != nil && len(podList.Items) == 0 {
		klog.V(1).Infof("No pod found for labelselector %s", labelSelector)
	}
	return nil, podList
}

func DeletePod(opt TestOptions, isHub bool, namespace, name string) error {
	clientKube := getKubeClient(opt, isHub)
	err := clientKube.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("Failed to delete pod %s in namespace %s due to %v", name, namespace, err)
		return err
	}
	return nil
}
