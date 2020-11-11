package utils

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"k8s.io/klog"
)

func GetGrafanaURL(opt TestOptions) string {
	grafanaConsoleURL := "https://multicloud-console.apps." + opt.HubCluster.BaseDomain + "/grafana/"
	if opt.HubCluster.GrafanaURL != "" {
		grafanaConsoleURL = opt.HubCluster.GrafanaURL
	} else {
		opt.HubCluster.GrafanaHost = "multicloud-console.apps." + opt.HubCluster.BaseDomain
	}
	return grafanaConsoleURL
}

func CheckGrafanaConsole(opt TestOptions) error {
	grafanaConsoleURL := GetGrafanaURL(opt)
	req, err := http.NewRequest("GET", grafanaConsoleURL, nil)
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	token, err := FetchBearerToken(opt)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Host = opt.HubCluster.GrafanaHost

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("resp.StatusCode: %v\n", resp.StatusCode)
		return fmt.Errorf("Failed to access grafana console")
	}
	return nil
}
