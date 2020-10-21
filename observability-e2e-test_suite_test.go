package main_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/open-cluster-management/observability-e2e-test/utils"
	"github.com/sclevine/agouti"
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

func TestOpenClusterManagementE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(reportFile)
	RunSpecsWithDefaultAndCustomReporters(t, "MCO_E2E Suite", []Reporter{junitReporter})
}

var agoutiDriver *agouti.WebDriver

var _ = BeforeSuite(func() {
	initVars()
})

var _ = AfterSuite(func() {
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
	// identity provider can either be 0 or 1
	// with 0 for kube:admin or `kubeadmin` and
	// 1 for any other use, ie. user defined users
	// default to `kubeadmin` logins, otherwise
	// select the second option
	testIdentityProvider = 0
	if kubeadminUser != "kubeadmin" {
		testIdentityProvider = 1
	}

	for i := range testOptions.ManagedClusters {
		if testOptions.ManagedClusters[i].MasterURL == "" {
			testOptions.ManagedClusters[i].MasterURL = fmt.Sprintf("https://api.%s:6443", testOptions.ManagedClusters[0].BaseDomain)
		}
	}

	if testOptions.ImageRegistry.Server != "" {
		registry = testOptions.ImageRegistry.Server
		registryUser = testOptions.ImageRegistry.User
		registryPassword = testOptions.ImageRegistry.Password
	} else {
		klog.Warningf("No `imageRegistry.server` was included in the options.yaml file. Ignoring any tests that require an ImageRegistry.")
	}

	//Allow developers and testers to use a non-default namespace for ACM deployment
	if testOptions.HubCluster.Namespace != "" {
		hubNamespace = testOptions.HubCluster.Namespace
		klog.Warningf("The non-default namespace used to deploy the ACM hub will be %s.  As specified by `hub.namespace` in the options.yaml file.", hubNamespace)
	} else {
		hubNamespace = "open-cluster-management"
	}
}
