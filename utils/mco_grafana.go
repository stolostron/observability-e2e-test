package utils

import (
	"crypto/tls"
	"errors"
	"net/http"
)

func CheckGrafanaConsole(opt TestOptions) error {
	req, err := http.NewRequest("GET", "https://multicloud-console.apps."+opt.HubCluster.BaseDomain+"/grafana/", nil)
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

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("Failed to access grafana console")
	}
	return nil
}
