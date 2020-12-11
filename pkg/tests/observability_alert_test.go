package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/slack-go/slack"

	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/open-cluster-management/observability-e2e-test/pkg/kustomize"
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
	statefulset := [...]string{"alertmanager", "observability-observatorium-thanos-rule"}
	configmap := [...]string{"thanos-ruler-default-rules", "thanos-ruler-custom-rules"}
	secret := "alertmanager-config"

	It("[P1,Sev1,observability]should have the expected statefulsets (alert/g0)", func() {
		By("Checking if STS: Alertmanager and observability-observatorium-thanos-rule exist")
		for _, name := range statefulset {
			sts, err := hubClient.AppsV1().StatefulSets(MCO_NAMESPACE).Get(name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(sts.Spec.Template.Spec.Volumes)).Should(BeNumerically(">", 0))

			if sts.GetName() == "alertmanager" {
				By("The statefulset: " + sts.GetName() + " should have the appropriate secret mounted")
				Expect(sts.Spec.Template.Spec.Volumes[0].Secret.SecretName).To(Equal("alertmanager-config"))
			}

			if sts.GetName() == "observability-observatorium-thanos-rule" {
				By("The statefulset: " + sts.GetName() + " should have the appropriate configmap mounted")
				Expect(sts.Spec.Template.Spec.Volumes[0].ConfigMap.Name).To(Equal("thanos-ruler-default-rules"))
			}
		}
	})

	It("[P2,Sev2,observability]should have the expected configmap (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-default-rules is existed")
		cm, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(configmap[0], metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(cm.ResourceVersion).ShouldNot(BeEmpty())
		klog.V(3).Infof("Configmap %s does exist", configmap[0])
	})

	It("[P3,Sev3,observability]should not have the CM: thanos-ruler-custom-rules (alert/g0)", func() {
		By("Checking if CM: thanos-ruler-custom-rules not existed")
		_, err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Get(configmap[1], metav1.GetOptions{})

		if err == nil {
			err = fmt.Errorf("%s exist within the namespace env", configmap[1])
			Expect(err).NotTo(HaveOccurred())
		}

		Expect(err).To(HaveOccurred())
		klog.V(3).Infof("Configmap %s does not exist", configmap[1])
	})

	It("[P2,Sev2,observability]should have the expected secret (alert/g0)", func() {
		By("Checking if SECRETS: alertmanager-config is existed")
		secret, err := hubClient.CoreV1().Secrets(MCO_NAMESPACE).Get(secret, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.GetName()).To(Equal("alertmanager-config"))
		klog.V(3).Infof("Successfully got secret: %s", secret.GetName())
	})

	It("[P1,Sev1,observability]should have custom alert generated (alert/g0)", func() {
		By("Creating custom alert rules")
		yamlB, _ := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/alerts/custom_rules_valid"})
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		var labelName, labelValue string
		labels, _ := kustomize.GetLabels(yamlB)
		for labelName = range labels.(map[string]interface{}) {
			labelValue = labels.(map[string]interface{})[labelName].(string)
		}

		By("Checking alert generated")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, `ALERTS{`+labelName+`="`+labelValue+`"}`, "2m",
				[]string{`"__name__":"ALERTS"`, `"` + labelName + `":"` + labelValue + `"`})
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P1,Sev1,observability]should modify the SECRET: alertmanager-config (alert/g0)", func() {
		By("Editing the secret, we should be able to add the third partying tools integrations")
		secret := utils.CreateCustomAlertConfigYaml(testOptions.HubCluster.BaseDomain)

		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, secret)).NotTo(HaveOccurred())
		klog.V(3).Infof("Successfully modified the secret: alertmanager-config")
	})

	It("[P1,Sev1,observability]should have custom alert updated (alert/g0)", func() {
		By("Updating custom alert rules")
		yamlB, _ := kustomize.Render(kustomize.Options{KustomizationPath: "../../observability-gitops/alerts/custom_rules_invalid"})
		Expect(utils.Apply(testOptions.HubCluster.MasterURL, testOptions.KubeConfig, testOptions.HubCluster.KubeContext, yamlB)).NotTo(HaveOccurred())

		var labelName, labelValue string
		labels, _ := kustomize.GetLabels(yamlB)
		for labelName = range labels.(map[string]interface{}) {
			labelValue = labels.(map[string]interface{})[labelName].(string)
		}

		By("Checking alert generated")
		Eventually(func() error {
			err, _ := utils.ContainManagedClusterMetric(testOptions, `ALERTS{`+labelName+`="`+labelValue+`"}`, "1m",
				[]string{`"__name__":"ALERTS"`, `"` + labelName + `":"` + labelValue + `"`})
			return err
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(MatchError("Failed to find metric name from response"))
	})

	It("[P1,Sev1,observability]should verify that the alerts are created (alert/g0)", func() {
		By("Checking that alertmanager and thanos-rule pods are running")
		podList, err := hubClient.CoreV1().Pods(MCO_NAMESPACE).List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		for _, pod := range podList.Items {
			if strings.Contains(pod.GetName(), "alertmanager") || strings.Contains(pod.GetName(), "thanos-rule") {
				Eventually(func() error {
					p, err := hubClient.CoreV1().Pods(MCO_NAMESPACE).Get(pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					if string(p.Status.Phase) != "Running" {
						klog.V(3).Infof("%s is (%s)", p.GetName(), string(p.Status.Phase))
						return fmt.Errorf("%s is waiting to run", p.GetName())
					}

					Expect(string(p.Status.Phase)).To(Equal("Running"))
					klog.V(3).Infof("%s is (%s)", p.GetName(), string(p.Status.Phase))
					return nil
				}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
			}
		}

		By("Viewing the channel that will hold the alert notifications")
		slackAPI := slack.New("xoxb-2253118358-1363717104599-GwMY2cdUV5Z1OZRu23egTuyf")

		bot, err := slackAPI.GetBotInfo("B01F7TM3692")
		Expect(err).NotTo(HaveOccurred())
		Expect(bot.Name).Should(Equal("TestingObserv"))
		klog.V(3).Infof("Found slack bot: %s", bot.Name)

		channel, err := slackAPI.GetConversationInfo("C01B4EK1JH1", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(channel.Name).Should(Equal("team-observability-test"))
		klog.V(3).Infof("Found slack channel for testing: %s", channel.Name)

		history, err := slackAPI.GetConversationHistory(&slack.GetConversationHistoryParameters{ChannelID: "C01B4EK1JH1", Limit: 10})
		Expect(err).NotTo(HaveOccurred())
		Expect(history.Ok).Should(Equal(true))

		Expect(len(history.Messages)).Should(BeNumerically(">", 0))
		klog.V(3).Infof("Found slack messages")
		for _, msg := range history.Messages {
			klog.Info(msg.Attachments[0].Text)
		}

		alertNotFound := true

		Eventually(func() error {
			history, err := slackAPI.GetConversationHistory(&slack.GetConversationHistoryParameters{ChannelID: "C01B4EK1JH1", Limit: 10})
			Expect(err).NotTo(HaveOccurred())
			Expect(history.Ok).Should(Equal(true))

			for _, alert := range history.Messages {
				if strings.Contains(alert.Attachments[0].TitleLink, baseDomain) {
					klog.V(3).Infof("Viewing alert (%s): "+alert.Attachments[0].Text, alert.Timestamp)
					Expect(alert.Attachments[0].Title).Should(Equal("[FIRING] NodeOutOfMemory (warning)"))
					alertNotFound = false
				}
			}

			if alertNotFound {
				klog.V(3).Infoln("Waiting for targeted alert..")
				return fmt.Errorf("no new slack alerts has been created")
			}

			return nil
		}, EventuallyTimeoutMinute*5, EventuallyIntervalSecond*5).Should(Succeed())
	})

	It("[P2,Sev2,observability]should delete the created configmap (alert/g0)", func() {
		err := hubClient.CoreV1().ConfigMaps(MCO_NAMESPACE).Delete(configmap[1], &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		klog.V(3).Infof("Successfully deleted CM: thanos-ruler-custom-rules")
	})

	AfterEach(func() {
		utils.PrintAllMCOPodsStatus(testOptions)
		utils.PrintAllOBAPodsStatus(testOptions)
		testFailed = testFailed || CurrentGinkgoTestDescription().Failed
	})
})
