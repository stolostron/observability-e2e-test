package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-cluster-management/observability-e2e-test/pkg/utils"
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

	It("[P1,Sev1,observability] should revert any manual changes on metrics-collector deployment (endpoint_preserve/g0)", func() {
		By("Deleting metrics-collector deployment")
		var (
			err error
			dep *appv1.Deployment
		)

		Eventually(func() error {
			err, dep = utils.GetDeployment(testOptions, false, "metrics-collector-deployment", MCO_ADDON_NAMESPACE)
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())

		Eventually(func() error {
			err = utils.DeleteDeployment(testOptions, false, "metrics-collector-deployment", MCO_ADDON_NAMESPACE)
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())

		newDep := &appv1.Deployment{}
		Eventually(func() bool {
			err, newDep = utils.GetDeployment(testOptions, false, "metrics-collector-deployment", MCO_ADDON_NAMESPACE)
			if err == nil {
				if dep.ObjectMeta.ResourceVersion != newDep.ObjectMeta.ResourceVersion {
					return true
				}
			}
			return false
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Updating metrics-collector deployment")
		updateSaName := "test-serviceaccount"
		Eventually(func() error {
			err, newDep = utils.GetDeployment(testOptions, false, "metrics-collector-deployment", MCO_ADDON_NAMESPACE)
			if err != nil {
				return err
			}
			newDep.Spec.Template.Spec.ServiceAccountName = updateSaName
			err, newDep = utils.UpdateDeployment(testOptions, false, "metrics-collector-deployment", MCO_ADDON_NAMESPACE, newDep)
			return err
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(Succeed())
		Eventually(func() bool {
			err, revertDep := utils.GetDeployment(testOptions, false, "metrics-collector-deployment", MCO_ADDON_NAMESPACE)
			if err == nil {
				if revertDep.ObjectMeta.ResourceVersion != newDep.ObjectMeta.ResourceVersion &&
					revertDep.Spec.Template.Spec.ServiceAccountName != updateSaName {
					return true
				}
			}
			return false
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("[P1,Sev1,observability] should revert any manual changes on metrics-collector-view clusterolebinding (endpoint_preserve/g0)", func() {
		By("Deleting metrics-collector-view clusterolebinding")
		err, crb := utils.GetCRB(testOptions, false, "metrics-collector-view")
		Expect(err).ToNot(HaveOccurred())
		err = utils.DeleteCRB(testOptions, false, "metrics-collector-view")
		Expect(err).ToNot(HaveOccurred())
		newCrb := &rbacv1.ClusterRoleBinding{}
		Eventually(func() bool {
			err, newCrb = utils.GetCRB(testOptions, false, "metrics-collector-view")
			if err == nil {
				if crb.ObjectMeta.ResourceVersion != newCrb.ObjectMeta.ResourceVersion {
					return true
				}
			}
			return false
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(BeTrue())

		By("Updating metrics-collector-view clusterolebinding")
		updateSubName := "test-subject"
		newCrb.Subjects[0].Name = updateSubName
		err, _ = utils.UpdateCRB(testOptions, false, "metrics-collector-view", newCrb)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			err, revertCrb := utils.GetCRB(testOptions, false, "metrics-collector-view")
			if err == nil {
				if revertCrb.ObjectMeta.ResourceVersion != newCrb.ObjectMeta.ResourceVersion &&
					revertCrb.Subjects[0].Name != updateSubName {
					return true
				}
			}
			return false
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	It("[P1,Sev1,observability] should recreate on metrics-collector-serving-certs-ca-bundle configmap if deleted (endpoint_preserve/g0)", func() {
		By("Deleting metrics-collector-serving-certs-ca-bundle configmap")
		var (
			err error
			cm  *v1.ConfigMap
		)

		Eventually(func() error {
			err, cm = utils.GetConfigMap(testOptions, false, "metrics-collector-serving-certs-ca-bundle", MCO_ADDON_NAMESPACE)
			return err
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(Succeed())

		Eventually(func() error {
			err = utils.DeleteConfigMap(testOptions, false, "metrics-collector-serving-certs-ca-bundle", MCO_ADDON_NAMESPACE)
			return err
		}, EventuallyTimeoutMinute*3, EventuallyIntervalSecond*5).Should(Succeed())

		Expect(err).ToNot(HaveOccurred())
		newCm := &v1.ConfigMap{}
		Eventually(func() bool {
			err, newCm = utils.GetConfigMap(testOptions, false, "metrics-collector-serving-certs-ca-bundle", MCO_ADDON_NAMESPACE)
			if err == nil {
				if cm.ObjectMeta.ResourceVersion != newCm.ObjectMeta.ResourceVersion {
					return true
				}
			}
			return false
		}, EventuallyTimeoutMinute*1, EventuallyIntervalSecond*5).Should(BeTrue())
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
