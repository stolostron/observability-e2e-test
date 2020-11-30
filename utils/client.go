package utils

import "k8s.io/client-go/kubernetes"

func getKubeClient(opt TestOptions, isHub bool) kubernetes.Interface {
	clientKube := NewKubeClient(
		opt.HubCluster.MasterURL,
		opt.KubeConfig,
		opt.HubCluster.KubeContext)
	if !isHub && len(opt.ManagedClusters) > 0 {
		clientKube = NewKubeClient(
			opt.ManagedClusters[0].MasterURL,
			opt.ManagedClusters[0].KubeConfig,
			"")
		// use the default context as workaround
		//			opt.ManagedClusters[0].KubeContext)
	}
	return clientKube
}
