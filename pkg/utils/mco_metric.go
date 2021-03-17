// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/klog"
)

func GetPrometheusURL(opt TestOptions) string {
	prometheusURL := "https://prometheus-k8s-openshift-monitoring.apps." + opt.HubCluster.BaseDomain
	return prometheusURL
}

func GetPrometheusMetricsMetadata(opt TestOptions) (error, []string) {
	prometheusURL := GetPrometheusURL(opt)
	path := "/api/v1/metadata"
	klog.V(1).Infof("request url is: %s\n", prometheusURL+path)
	req, err := http.NewRequest(
		"GET",
		prometheusURL+path,
		nil)
	if err != nil {
		return err, nil
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	token, err := FetchBearerToken(opt)
	if err != nil {
		return err, nil
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Host = opt.HubCluster.GrafanaHost

	resp, err := client.Do(req)
	if err != nil {
		return err, nil
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("resp.StatusCode: %v\n", resp.StatusCode)
		return fmt.Errorf("Failed to access prometheus metrics metadata"), nil
	}

	metricResult, err := ioutil.ReadAll(resp.Body)
	klog.V(1).Infof("queryResult: %s\n", metricResult)
	if err != nil {
		return err, nil
	}

	if !strings.Contains(string(metricResult), `"status":"success"`) {
		return fmt.Errorf("Failed to find valid status from response"), nil
	}

	metaData := make(map[string]interface{})

	err = json.Unmarshal([]byte(metricResult), &metaData)

	data := metaData["data"].(map[string]interface{})

	names := make([]string, 0, len(data))
	for k := range data {
		names = append(names, k)
	}

	return nil, names
}

func ContainManagedClusterMetric(opt TestOptions, query string, matchedLabels []string) (error, bool) {
	grafanaConsoleURL := GetGrafanaURL(opt)
	path := "/api/datasources/proxy/1/api/v1/query?"
	queryParams := url.PathEscape(fmt.Sprintf("query=%s", query))
	klog.V(1).Infof("request url is: %s\n", grafanaConsoleURL+path+queryParams)
	req, err := http.NewRequest(
		"GET",
		grafanaConsoleURL+path+queryParams,
		nil)
	if err != nil {
		return err, false
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	token, err := FetchBearerToken(opt)
	if err != nil {
		return err, false
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Host = opt.HubCluster.GrafanaHost

	resp, err := client.Do(req)
	if err != nil {
		return err, false
	}

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("resp.StatusCode: %v\n", resp.StatusCode)
		return fmt.Errorf("Failed to access managed cluster metrics via grafana console"), false
	}

	metricResult, err := ioutil.ReadAll(resp.Body)
	klog.V(1).Infof("metricResult: %s\n", metricResult)
	if err != nil {
		return err, false
	}

	if !strings.Contains(string(metricResult), `"status":"success"`) {
		return fmt.Errorf("Failed to find valid status from response"), false
	}

	contained := true
	for _, label := range matchedLabels {
		if !strings.Contains(string(metricResult), label) {
			contained = false
			break
		}
	}
	if !contained {
		return fmt.Errorf("Failed to find metric name from response"), false
	}

	return nil, true
}
