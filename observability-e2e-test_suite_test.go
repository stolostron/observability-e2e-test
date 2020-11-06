package main_test

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/utils"
)

//var bareBaseDomain string
var baseDomain string
var kubeadminUser string
var kubeadminCredential string
var kubeconfig string
var reportFile string

var registry string
var registryUser string
var registryPassword string

var optionsFile, clusterDeployFile, installConfigFile string
var testOptions utils.TestOptions
var clusterDeploy utils.ClusterDeploy
var installConfig utils.InstallConfig
var testOptionsContainer utils.TestOptionsContainer
var testUITimeout time.Duration
var testHeadless bool
var testIdentityProvider int

var ownerPrefix string

var hubNamespace string
var pullSecretName string
var installConfigAWS, installConfigGCP, installConfigAzure string
var hiveClusterName, hiveGCPClusterName, hiveAzureClusterName string

var ocpRelease string

const OCP_RELEASE_DEFAULT = "4.4.4"

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randString(length int) string {
	return StringWithCharset(length, charset)
}

func init() {
	klog.SetOutput(GinkgoWriter)
	klog.InitFlags(nil)

	flag.StringVar(&kubeadminUser, "kubeadmin-user", "kubeadmin", "Provide the kubeadmin credential for the cluster under test (e.g. -kubeadmin-user=\"xxxxx\").")
	flag.StringVar(&kubeadminCredential, "kubeadmin-credential", "", "Provide the kubeadmin credential for the cluster under test (e.g. -kubeadmin-credential=\"xxxxx-xxxxx-xxxxx-xxxxx\").")
	flag.StringVar(&baseDomain, "base-domain", "", "Provide the base domain for the cluster under test (e.g. -base-domain=\"demo.red-chesterfield.com\").")
	flag.StringVar(&reportFile, "report-file", "results.xml", "Provide the path to where the junit results will be printed.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Location of the kubeconfig to use; defaults to KUBECONFIG if not set")
	flag.StringVar(&optionsFile, "options", "", "Location of an \"options.yaml\" file to provide input for various tests")
}

func TestObservabilityE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(reportFile)
	RunSpecsWithDefaultAndCustomReporters(t, "Observability E2E Suite", []Reporter{junitReporter})
}

var agoutiDriver *agouti.WebDriver

var _ = BeforeSuite(func() {
	initVars()

	if os.Getenv("SKIP_INSTALL_STEP") != "true" {
		hubClient := utils.NewKubeClient(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)

		dynClient := utils.NewKubeClientDynamic(
			testOptions.HubCluster.MasterURL,
			testOptions.KubeConfig,
			testOptions.HubCluster.KubeContext)

		EventuallyTimeoutMinute := 60 * time.Second
		EventuallyIntervalSecond := 1 * time.Second

		By("Checking MCO operator is existed")
		var podList, _ = hubClient.CoreV1().Pods(MCO_OPERATOR_NAMESPACE).List(metav1.ListOptions{LabelSelector: MCO_LABEL})
		Expect(len(podList.Items)).To(Equal(1))
		for _, pod := range podList.Items {
			Expect(string(pod.Status.Phase)).To(Equal("Running"))
		}

		By("Checking Required CRDs is existed")
		Eventually(func() error {
			return utils.HaveCRDs(testOptions.HubCluster, testOptions.KubeConfig,
				[]string{
					"multiclusterobservabilities.observability.open-cluster-management.io",
					"observatoria.core.observatorium.io",
					"observabilityaddons.observability.open-cluster-management.io",
				})
		}).Should(Succeed())

		Expect(utils.CreateMCONamespace(testOptions)).NotTo(HaveOccurred())
		Expect(utils.CreatePullSecret(testOptions)).NotTo(HaveOccurred())
		Expect(utils.CreateObjSecret(testOptions)).NotTo(HaveOccurred())

		By("Creating MCO instance")
		mco := utils.NewMCOInstanceYaml(MCO_CR_NAME)
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, mco)).NotTo(HaveOccurred())

		By("Waiting for MCO ready status")
		Eventually(func() bool {
			instance, err := dynClient.Resource(utils.NewMCOGVR()).Get(MCO_CR_NAME, metav1.GetOptions{})
			if err == nil {
				return utils.StatusContainsTypeEqualTo(instance, "Ready")
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())
	}
})

var _ = AfterSuite(func() {
	hubClient := utils.NewKubeClient(
		testOptions.HubCluster.MasterURL,
		testOptions.KubeConfig,
		testOptions.HubCluster.KubeContext)

	dynClient := utils.NewKubeClientDynamic(
		testOptions.HubCluster.MasterURL,
		testOptions.KubeConfig,
		testOptions.HubCluster.KubeContext)

	EventuallyTimeoutMinute := 60 * time.Second
	EventuallyIntervalSecond := 1 * time.Second

	By("Uninstall MCO instance")
	err := utils.UninstallMCO(testOptions)
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for delete all MCO components")
	Eventually(func() error {
		var podList, _ = hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
		if len(podList.Items) != 0 {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Waiting for delete MCO addon instance")
	Eventually(func() error {
		gvr := utils.NewMCOAddonGVR()
		name := MCO_CR_NAME + "-addon"
		instance, _ := dynClient.Resource(gvr).Namespace("local-cluster").Get(name, metav1.GetOptions{})
		if instance != nil {
			return errors.New("Failed to delete MCO addon instance")
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Waiting for delete all MCO addon components")
	Eventually(func() error {
		var podList, _ = hubClient.CoreV1().Pods(MCO_ADDON_NAMESPACE).List(metav1.ListOptions{})
		if len(podList.Items) != 0 {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())

	By("Waiting for delete MCO namespaces")
	Eventually(func() error {
		err := hubClient.CoreV1().Namespaces().Delete(MCO_NAMESPACE, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		return nil
	}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
})

func initVars() {

	// default ginkgo test timeout 30s
	// increased from original 10s
	testUITimeout = time.Second * 30

	if optionsFile == "" {
		optionsFile = os.Getenv("OPTIONS")
		if optionsFile == "" {
			optionsFile = "resources/options.yaml"
		}
	}

	klog.V(1).Infof("options filename=%s", optionsFile)

	data, err := ioutil.ReadFile(optionsFile)
	if err != nil {
		klog.Errorf("--options error: %v", err)
	}
	Expect(err).NotTo(HaveOccurred())

	fmt.Printf("file preview: %s \n", string(optionsFile))

	err = yaml.Unmarshal([]byte(data), &testOptionsContainer)
	if err != nil {
		klog.Errorf("--options error: %v", err)
	}

	testOptions = testOptionsContainer.Options

	// default Headless is `true`
	// to disable, set Headless: false
	// in options file
	if testOptions.Headless == "" {
		testHeadless = true
	} else {
		if testOptions.Headless == "false" {
			testHeadless = false
		} else {
			testHeadless = true
		}
	}

	// OwnerPrefix is used to help identify who owns deployed resources
	//    If a value is not supplied, the default is OS environment variable $USER
	if testOptions.OwnerPrefix == "" {
		ownerPrefix = os.Getenv("USER")
		if ownerPrefix == "" {
			ownerPrefix = "ginkgo"
		}
	} else {
		ownerPrefix = testOptions.OwnerPrefix
	}
	klog.V(1).Infof("ownerPrefix=%s", ownerPrefix)

	if testOptions.Connection.OCPRelease == "" {
		ocpRelease = OCP_RELEASE_DEFAULT
	} else {
		ocpRelease = testOptions.Connection.OCPRelease
	}
	klog.V(1).Infof("ocpRelease=%s", ocpRelease)

	if testOptions.KubeConfig == "" {
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		testOptions.KubeConfig = kubeconfig
	}

	if testOptions.HubCluster.BaseDomain != "" {
		baseDomain = testOptions.HubCluster.BaseDomain

		if testOptions.HubCluster.MasterURL == "" {
			testOptions.HubCluster.MasterURL = fmt.Sprintf("https://api.%s:6443", testOptions.HubCluster.BaseDomain)
		}

	} else {
		klog.Warningf("No `hub.baseDomain` was included in the options.yaml file. Tests will be unable to run. Aborting ...")
		Expect(testOptions.HubCluster.BaseDomain).NotTo(BeEmpty(), "The `hub` option in options.yaml is required.")
	}
	if testOptions.HubCluster.User != "" {
		kubeadminUser = testOptions.HubCluster.User
	}
	if testOptions.HubCluster.Password != "" {
		kubeadminCredential = testOptions.HubCluster.Password
	}
}
