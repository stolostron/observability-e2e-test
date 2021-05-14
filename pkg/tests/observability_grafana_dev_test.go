// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"bytes"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

var _ = Describe("Observability:", func() {

	It("[P1][Sev1][Observability][Integration] Should run grafana-dev test successfully (grafana-dev/g0)", func() {
		Eventually(func() error {
			cmd := exec.Command("../../cicd-scripts/grafana-dev-test.sh")
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				klog.V(1).Infof("Failed to run grafana-dev-test.sh: %v", out.String())
			}
			return err
		}, EventuallyTimeoutMinute*10, EventuallyIntervalSecond*5).Should(Succeed())
	})

	AfterEach(func() {
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
		if testFailed {
			utils.PrintMCORelatedInfoForDebug(testOptions)
		} else {
			Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
		}
	})
})
