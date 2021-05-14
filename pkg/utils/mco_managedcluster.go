// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func UpdateObservabilityFromManagedCluster(opt TestOptions, enableObservability bool) error {
	clusterName := GetManagedClusterName(opt)
	if clusterName != "" {
		clientDynamic := GetKubeClientDynamic(opt, true)
		cluster, err := clientDynamic.Resource(NewOCMManagedClustersGVR()).Get(clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		labels, ok := cluster.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
		if !ok {
			cluster.Object["metadata"].(map[string]interface{})["labels"] = map[string]interface{}{}
			labels = cluster.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
		}

		if !enableObservability {
			labels["observability"] = "disabled"
		} else {
			delete(labels, "observability")
		}
		_, updateErr := clientDynamic.Resource(NewOCMManagedClustersGVR()).Update(cluster, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
	}
	return nil
}

func PrintManagedCluster(opt TestOptions) {
	clusterName := GetManagedClusterName(opt)
	if clusterName != "" {
		clientDynamic := GetKubeClientDynamic(opt, true)
		cluster, err := clientDynamic.Resource(NewOCMManagedClustersGVR()).Get(clusterName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed to get managedcluster %+v: %v", clusterName, err)
			return
		}
		klog.V(1).Infof("managedcluster <%v>: %+v\n", clusterName, cluster)
	}
	klog.Errorf("Failed to found managedcluster")
}
