package main_test

import (
	"fmt"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-cluster-management/observability-e2e-test/utils"

	//. "github.com/sclevine/agouti/matchers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	MANAGEMENT_INGRESS_POD_NAMESPACE = "open-cluster-management"
	MANAGEMENT_INGRESS_DEPLOY_PREFIX = "multicluster-observability-operator"
	MANAGEMENT_INGRESS_LABEL         = "name=multicluster-observability-operator"
)

var _ = Describe("MCO Operator testing", func() {
	var hubClient kubernetes.Interface
	BeforeEach(func() {
		//fmt.Printf("\n\nConnecting to the Hub with master-url: %s\n\tcontext: %s\n\tfrom kubeconfig: %s\n\n", testOptions.HubCluster.MasterURL, testOptions.HubCluster.KubeContext, testOptions.KubeConfig)
		io.WriteString(GinkgoWriter, fmt.Sprintf("\n\nConnecting to the Hub with master-url: %s\n\tcontext: %s\n\tfrom kubeconfig: %s\n\n", testOptions.HubCluster.MasterURL, testOptions.HubCluster.KubeContext, testOptions.KubeConfig))
		hubClient = utils.NewKubeClient(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext)
	})

	It("should have the expected running mco operator in namespace: open-cluster-management (ingress/g0)", func() {
		var podList, _ = hubClient.CoreV1().Pods(MANAGEMENT_INGRESS_POD_NAMESPACE).List(metav1.ListOptions{LabelSelector: MANAGEMENT_INGRESS_LABEL})
		//io.WriteString(GinkgoWriter, fmt.Sprintf("\n\nPod details: %s\n\tcontext: \n\n", podList.String()))

		Expect(len(podList.Items)).To(Equal(1))
		for _, pod := range podList.Items {
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
		}
	})

	/* It("should allow the user to login to web console (ingress/g0)", func() {
		console := "https://multicloud-console.apps." + baseDomain + "/multicloud/"
		login := "https://oauth-openshift.apps." + baseDomain + "/login"
		defaultOptions := []string{
			"ignore-certificate-errors",
			"disable-gpu",
			"no-sandbox",
		}

		if testHeadless {
			defaultOptions = append(defaultOptions, "headless")
		}

		page, err := agoutiDriver.NewPage(agouti.Desired(agouti.Capabilities{
			"chromeOptions": map[string][]string{
				"args": defaultOptions,
			},
		}))
		Expect(err).NotTo(HaveOccurred())
		SetDefaultEventuallyTimeout(testUITimeout)

		By("redirecting the user to the OpenShift login form", func() {
			Expect(page.Navigate(console)).To(Succeed())
			if Expect(page.URL()).To(ContainSubstring("/oauth")) {
				page.AllByClass("idp").At(testIdentityProvider).Click()
			}
			Expect(page.URL()).To(HavePrefix(login))
		})

		By("allowing the user to fill out the login form and submit it", func() {
			Eventually(page.FindByID("inputUsername")).Should(BeFound())
			Eventually(page.FindByID("inputPassword")).Should(BeFound())
			Expect(page.FindByID("inputUsername").Fill(kubeadminUser)).To(Succeed())
			Expect(page.FindByID("inputPassword").Fill(kubeadminCredential)).To(Succeed())
			_, err := page.FindByClass("form-horizontal").Active()
			if err != nil {
				Expect(page.FindByClass("pf-c-form").Submit()).To(Succeed())
			} else {
				Expect(page.FindByClass("form-horizontal").Submit()).To(Succeed())
			}
		})
		Expect(page.Destroy()).To(Succeed())
	}) */
})
