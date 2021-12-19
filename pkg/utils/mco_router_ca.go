// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package utils

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	RouterCertsSecretName = "custom-cert"
)

func GetRouterCA(cli kubernetes.Interface) ([]byte, error) {
	var caCrt []byte
	caSecret, err := cli.CoreV1().
		Secrets("openshift-ingress").
		Get(RouterCertsSecretName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get router certificate secret %s due to %v", RouterCertsSecretName, err)
		return caCrt, err
	}
	caCrt, ok := caSecret.Data["tls.crt"]
	if ok {
		return caCrt, nil
	}
	return caCrt, fmt.Errorf("failed to get tls.crt from %s secret", RouterCertsSecretName)
}
