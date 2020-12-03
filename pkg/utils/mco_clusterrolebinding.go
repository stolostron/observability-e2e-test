package utils

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func GetCRB(opt TestOptions, isHub bool, name string) (error, *rbacv1.ClusterRoleBinding) {
	clientKube := getKubeClient(opt, isHub)
	crb, err := clientKube.RbacV1().ClusterRoleBindings().Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get cluster rolebinding %s due to %v", name, err)
	}
	return err, crb
}

func DeleteCRB(opt TestOptions, isHub bool, name string) error {
	clientKube := getKubeClient(opt, isHub)
	err := clientKube.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("Failed to delete cluster rolebinding %s due to %v", name, err)
	}
	return err
}

func UpdateCRB(opt TestOptions, isHub bool, name string,
	crb *rbacv1.ClusterRoleBinding) (error, *rbacv1.ClusterRoleBinding) {
	clientKube := getKubeClient(opt, isHub)
	updateCRB, err := clientKube.RbacV1().ClusterRoleBindings().Update(crb)
	if err != nil {
		klog.Errorf("Failed to update cluster rolebinding %s due to %v", name, err)
	}
	return err, updateCRB
}
