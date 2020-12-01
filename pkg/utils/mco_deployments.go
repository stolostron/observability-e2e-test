package utils

import (
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func GetDeployment(opt TestOptions, isHub bool, name string,
	namespace string) (error, *appv1.Deployment) {
	clientKube := getKubeClient(opt, isHub)
	dep, err := clientKube.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get deployment %s in namespace %s due to %v", name, namespace, err)
	}
	return err, dep
}

func DeleteDeployment(opt TestOptions, isHub bool, name string, namespace string) error {
	clientKube := getKubeClient(opt, isHub)
	err := clientKube.AppsV1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("Failed to delete deployment %s in namespace %s due to %v", name, namespace, err)
	}
	return err
}

func UpdateDeployment(opt TestOptions, isHub bool, name string, namespace string,
	dep *appv1.Deployment) (error, *appv1.Deployment) {
	clientKube := getKubeClient(opt, isHub)
	updateDep, err := clientKube.AppsV1().Deployments(namespace).Update(dep)
	if err != nil {
		klog.Errorf("Failed to update deployment %s in namespace %s due to %v", name, namespace, err)
	}
	return err, updateDep
}
