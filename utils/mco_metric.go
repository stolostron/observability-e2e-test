package utils

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

func ContainManagedClusterMetric(opt TestOptions) (error, bool) {
	path := "/grafana/api/datasources/proxy/1/api/v1/"
	queryParams := "query?query=cluster%3Acapacity_cpu_cores%3Asum"
	req, err := http.NewRequest(
		"GET",
		"https://multicloud-console.apps."+opt.HubCluster.BaseDomain+path+queryParams,
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

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err, false
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("Failed to access managed cluster metrics via grafana console"), false
	}

	metricResult, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, false
	}

	if !strings.Contains(string(metricResult), `status":"success"`) {
		return errors.New("Failed to find valid status from response"), false
	}

	if !strings.Contains(string(metricResult), `"__name__":"cluster:capacity_cpu_cores:sum"`) {
		return errors.New("Failed to find metric name from response"), false
	}

	return nil, true
}
