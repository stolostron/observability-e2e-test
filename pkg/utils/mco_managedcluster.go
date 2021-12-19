// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package utils

import (
	"encoding/json"
	"strings"

	goversion "github.com/hashicorp/go-version"
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

func ListOCPManagedClusterIDs(opt TestOptions, minVersionStr string) ([]string, error) {
	minVersion, err := goversion.NewVersion(minVersionStr)
	if err != nil {
		return nil, err
	}
	clientDynamic := GetKubeClientDynamic(opt, true)
	objs, err := clientDynamic.Resource(NewOCMManagedClustersGVR()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	klog.V(3).Infof("objs is %s", objs)
	clusterIDs := []string{}
	for _, obj := range objs.Items {
		metadata := obj.Object["metadata"].(map[string]interface{})
		klog.V(3).Infof("metadata is %s", metadata)
		labels := metadata["labels"].(map[string]interface{})
		klog.V(3).Infof("labels is %s", labels)
		if labels != nil {
			vendorStr := ""
			if vendor, ok := labels["vendor"]; ok {
				vendorStr = vendor.(string)
			}

			clusterNameStr := ""
			if clusterName, ok := labels["name"]; ok {
				clusterNameStr = clusterName.(string)
			}
			klog.V(3).Infof("start to get obaAddon")
			addon, err := clientDynamic.Resource(NewMCOAddonGVR()).Namespace(clusterNameStr).Get("observability-addon", metav1.GetOptions{})
			klog.V(3).Infof("addon is %s ", addon)
			if err != nil {
				return nil, err
			}

			status, _ := json.MarshalIndent(addon.Object["status"], "", "  ")
			klog.V(3).Infof("status is %s ", status)
			obsAddonStatusStr := ""
			if strings.Contains(string(status), "Cluster metrics sent successfully") {
				obsAddonStatusStr = "available"
			}

			if vendorStr == "OpenShift" && obsAddonStatusStr == "available" {
				clusterVersionStr := ""
				if clusterVersionVal, ok := labels["openshiftVersion"]; ok {
					clusterVersionStr = clusterVersionVal.(string)
				}
				clusterVersion, err := goversion.NewVersion(clusterVersionStr)
				if err != nil {
					return nil, err
				}
				if clusterVersion.GreaterThanOrEqual(minVersion) {
					clusterIDStr := ""
					if clusterID, ok := labels["clusterID"]; ok {
						clusterIDStr = clusterID.(string)
					}
					if len(clusterIDStr) > 0 {
						clusterIDs = append(clusterIDs, clusterIDStr)
					}
				}
			}
		}
	}

	return clusterIDs, nil
}
