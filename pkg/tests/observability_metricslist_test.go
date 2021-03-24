// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

const (
	allowlistCMname = "observability-metrics-custom-allowlist"
)

var _ = Describe("Observability:", func() {
	BeforeEach(func() {
		hubClient = utils.NewKubeClient(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)

		dynClient = utils.NewKubeClientDynamic(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)
	})

	It("[P1][Sev1][Observability] Should have metrics which defined in custom metrics allowlist (metricslist/g0)", func() {
		By("Adding custom metrics allowlist configmap")
		yamlB, err := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/metrics/allowlist"})
		Expect(err).ToNot(HaveOccurred())
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		By("Waiting for new added metrics on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes offset 1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1][Sev1][Observability] Should have no metrics after custom metrics allowlist deleted (metricslist/g0)", func() {
		By("Deleting custom metrics allowlist configmap")
		Eventually(func() error {
			err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(allowlistCMname, &metav1.DeleteOptions{})
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*1).Should(Succeed())

		By("Waiting for new added metrics disappear on grafana console")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, "node_memory_Active_bytes offset 1m", []string{`"__name__":"node_memory_Active_bytes"`})
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})

	It("[P1][Sev1][Observability] Should have metrics which defined in metrics allowlist (metricslist/g0)", func() {
		Skip("Skip the test for default metrics allowlist")
		runDuration := 30
		runCount := 30

		// Workaround for https://github.com/open-cluster-management/backlog/issues/10481
		By("Getting metrics list from prometheus metadata")
		err, prometheusMetrics := utils.GetPrometheusMetricsMetadata(testOptions)

		By("Getting metrics allowlist from obs configmap")
		allowlistName := "observability-metrics-allowlist"
		cm, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(allowlistName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		ml := cm.Data["metrics_list.yaml"]
		data := make(map[string][]string)
		err = yaml.Unmarshal([]byte(ml), &data)
		obsMetrics := data["names"]

		By("Get the intersection of two metrics list")
		names := []string{}
		for i, v := range obsMetrics {
			if v == "" {
				continue
			}
			match := false
			for _, w := range prometheusMetrics {
				if v == w {
					match = true
				}
			}
			if match {
				names = append(names, obsMetrics[i])
			}
		}

		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

		By("Get metrics data")
		startTime := time.Now().Unix()
		for i, name := range names {
			if i >= runCount {
				break
			}
			if (time.Now().Unix() - startTime) > int64(runDuration) {
				klog.V(1).Infof(fmt.Sprintf("Over %d seconds", runDuration))
				break
			}
			klog.V(1).Infof("Getting metrics data: " + strconv.Itoa(i) + " => " + name)
			Eventually(func() error {
				err, _ := utils.ContainManagedClusterMetric(testOptions, name, []string{fmt.Sprintf(`"__name__":"%s"`, name)})
				return err
			}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
		}
	})

	AfterEach(func() {
		if testFailed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
