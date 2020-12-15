package utils

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func UpdateObservabilityFromManagedCluster(opt TestOptions, enableObservability bool) error {
	clientDynamic := getKubeClientDynamic(opt, true)
	clusters, err := clientDynamic.Resource(NewOCMManagedClustersGVR()).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cluster := range clusters.Items {
		labels := cluster.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
		if !enableObservability {
			labels["observability"] = "disabled"
		} else {
			delete(labels, "observability")
		}
		_, updateErr := clientDynamic.Resource(NewOCMManagedClustersGVR()).Update(&cluster, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
	}
	return nil
}
