module github.com/stolostron/observability-e2e-test

go 1.14

require (
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/hashicorp/go-version v1.3.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/alertmanager v0.23.0
	github.com/prometheus/common v0.30.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d // indirect
	golang.org/x/sys v0.0.0-20210915083310-ed5796bab164 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.17.2
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/kustomize/api v0.8.8
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/prometheus/common => github.com/prometheus/common v0.26.0
