package utils

func GetGrafanaURL(opt TestOptions) string {
	grafanaConsoleURL := "https://multicloud-console.apps." + opt.HubCluster.BaseDomain + "/grafana/"
	if opt.HubCluster.GrafanaURL != "" {
		grafanaConsoleURL = opt.HubCluster.GrafanaURL
	} else {
		opt.HubCluster.GrafanaHost = "multicloud-console.apps." + opt.HubCluster.BaseDomain
	}
	return grafanaConsoleURL
}
