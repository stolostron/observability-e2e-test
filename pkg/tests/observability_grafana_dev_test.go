// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tests

import (
	"bytes"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
)

var _ = Describe("Observability:", func() {

	// Do not need to run this case in canary environment
	// If we really need it in canary, ensure the grafana-dev-test.sh is available in observability-e2e-test image and all required commands exist
	It("[P1][Sev1][Observability][Integration] Should run grafana-dev test successfully (grafana-dev/g0)", func() {
		cmd := exec.Command("../../cicd-scripts/grafana-dev-test.sh")
		var out bytes.Buffer
		cmd.Stdout = &out
		Expect(cmd.Run()).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			utils.PrintMCOObject(testOptions)
			utils.PrintAllMCOPodsStatus(testOptions)
			utils.PrintAllOBAPodsStatus(testOptions)
		} else {
			Expect(utils.IntegrityChecking(testOptions)).NotTo(HaveOccurred())
		}
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
