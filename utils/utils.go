package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/prometheus/common/log"

	"github.com/sclevine/agouti"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewUnversionedRestClient(url, kubeconfig, context string) *rest.RESTClient {
	klog.V(5).Infof("Create unversionedRestClient for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, context)
	if err != nil {
		panic(err)
	}

	oldNegotiatedSerializer := config.NegotiatedSerializer
	config.NegotiatedSerializer = unstructuredscheme.NewUnstructuredNegotiatedSerializer()
	kubeRESTClient, err := rest.UnversionedRESTClientFor(config)
	// restore cfg before leaving
	defer func(cfg *rest.Config) { cfg.NegotiatedSerializer = oldNegotiatedSerializer }(config)

	if err != nil {
		panic(err)
	}

	return kubeRESTClient
}

func NewKubeClient(url, kubeconfig, context string) kubernetes.Interface {
	klog.V(5).Infof("Create kubeclient for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, context)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func NewKubeClientDynamic(url, kubeconfig, context string) dynamic.Interface {
	klog.V(5).Infof("Create kubeclient dynamic for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, context)
	if err != nil {
		panic(err)
	}

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func NewKubeClientAPIExtension(url, kubeconfig, context string) apiextensionsclientset.Interface {
	klog.V(5).Infof("Create kubeclient apiextension for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, context)
	if err != nil {
		panic(err)
	}

	clientset, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

// func NewKubeClientDiscovery(url, kubeconfig, context string) *discovery.DiscoveryClient {
// 	klog.V(5).Infof("Create kubeclient discovery for url %s using kubeconfig path %s\n", url, kubeconfig)
// 	config, err := LoadConfig(url, kubeconfig, context)
// 	if err != nil {
// 		panic(err)
// 	}

// 	clientset, err := discovery.NewDiscoveryClientForConfig(config)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return clientset
// }

func LoadConfig(url, kubeconfig, context string) (*rest.Config, error) {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	klog.V(5).Infof("Kubeconfig path %s\n", kubeconfig)
	// If we have an explicit indication of where the kubernetes config lives, read that.
	if kubeconfig != "" {
		if context == "" {
			// klog.V(5).Infof("clientcmd.BuildConfigFromFlags with %s and %s", url, kubeconfig)
			return clientcmd.BuildConfigFromFlags(url, kubeconfig)
		} else {
			return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
				&clientcmd.ConfigOverrides{
					CurrentContext: context,
				}).ClientConfig()
		}
	}
	// If not, try the in-cluster config.
	if c, err := rest.InClusterConfig(); err == nil {
		// log.Print("incluster\n")
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory.
	if usr, err := user.Current(); err == nil {
		klog.V(5).Infof("clientcmd.BuildConfigFromFlags for url %s using %s\n", url, filepath.Join(usr.HomeDir, ".kube", "config"))
		if c, err := clientcmd.BuildConfigFromFlags(url, filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not create a valid kubeconfig")

}

//Apply a multi resources file to the cluster described by the url, kubeconfig and context.
//url of the cluster
//kubeconfig which contains the context
//context, the context to use
//yamlB, a byte array containing the resources file
func Apply(url string, kubeconfig string, context string, yamlB []byte) error {
	yamls := strings.Split(string(yamlB), "---")
	// yamlFiles is an []string
	for _, f := range yamls {
		if len(strings.TrimSpace(f)) == 0 {
			continue
		}

		obj := &unstructured.Unstructured{}
		klog.V(5).Infof("obj:%v\n", obj.Object)
		err := yaml.Unmarshal([]byte(f), obj)
		if err != nil {
			return err
		}

		var kind string
		if v, ok := obj.Object["kind"]; !ok {
			return fmt.Errorf("kind attribute not found in %s", f)
		} else {
			kind = v.(string)
		}

		klog.V(5).Infof("kind: %s\n", kind)

		clientKube := NewKubeClient(url, kubeconfig, context)
		clientAPIExtension := NewKubeClientAPIExtension(url, kubeconfig, context)
		// now use switch over the type of the object
		// and match each type-case
		switch kind {
		case "CustomResourceDefinition":
			klog.V(5).Infof("Install CRD: %s\n", f)
			obj := &apiextensionsv1beta1.CustomResourceDefinition{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientAPIExtension.ApiextensionsV1beta1().CustomResourceDefinitions().Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientAPIExtension.ApiextensionsV1beta1().CustomResourceDefinitions().Create(obj)
			} else {
				existingObject.Spec = obj.Spec
				klog.Warningf("CRD %s already exists, updating!", existingObject.Name)
				_, err = clientAPIExtension.ApiextensionsV1beta1().CustomResourceDefinitions().Update(existingObject)
			}
		case "Namespace":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &corev1.Namespace{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.CoreV1().Namespaces().Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().Namespaces().Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s already exists, updating!", obj.Kind, obj.Name)
				_, err = clientKube.CoreV1().Namespaces().Update(existingObject)
			}
		case "ServiceAccount":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &corev1.ServiceAccount{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.CoreV1().ServiceAccounts(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().ServiceAccounts(obj.Namespace).Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.CoreV1().ServiceAccounts(obj.Namespace).Update(obj)
			}
		case "ClusterRoleBinding":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &rbacv1.ClusterRoleBinding{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.RbacV1().ClusterRoleBindings().Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.RbacV1().ClusterRoleBindings().Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.RbacV1().ClusterRoleBindings().Update(obj)
			}
		case "Secret":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &corev1.Secret{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.CoreV1().Secrets(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().Secrets(obj.Namespace).Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.CoreV1().Secrets(obj.Namespace).Update(obj)
			}
		case "Service":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &corev1.Service{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.CoreV1().Services(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().Services(obj.Namespace).Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.CoreV1().Services(obj.Namespace).Update(obj)
			}
		case "PersistentVolumeClaim":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &corev1.PersistentVolumeClaim{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.CoreV1().PersistentVolumeClaims(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().PersistentVolumeClaims(obj.Namespace).Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.CoreV1().PersistentVolumeClaims(obj.Namespace).Update(obj)
			}

		case "Deployment":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &appsv1.Deployment{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.AppsV1().Deployments(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.AppsV1().Deployments(obj.Namespace).Create(obj)
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.AppsV1().Deployments(obj.Namespace).Update(obj)
			}

		default:
			switch kind {
			case "MultiClusterObservability":
				klog.V(5).Infof("Install MultiClusterObservability: %s\n", f)
			default:
				return fmt.Errorf("Resource %s not supported", kind)
			}

			gvr := NewMCOGVR()
			clientDynamic := NewKubeClientDynamic(url, kubeconfig, context)
			if ns := obj.GetNamespace(); ns != "" {
				existingObject, errGet := clientDynamic.Resource(gvr).Namespace(ns).Get(obj.GetName(), metav1.GetOptions{})
				if errGet != nil {
					_, err = clientDynamic.Resource(gvr).Namespace(ns).Create(obj, metav1.CreateOptions{})
				} else {
					obj.Object["metadata"] = existingObject.Object["metadata"]
					klog.Warningf("%s %s/%s already exists, updating!", obj.GetKind(), obj.GetNamespace(), obj.GetName())
					_, err = clientDynamic.Resource(gvr).Namespace(ns).Update(obj, metav1.UpdateOptions{})
				}
			} else {
				existingObject, errGet := clientDynamic.Resource(gvr).Get(obj.GetName(), metav1.GetOptions{})
				if errGet != nil {
					_, err = clientDynamic.Resource(gvr).Create(obj, metav1.CreateOptions{})
				} else {
					obj.Object["metadata"] = existingObject.Object["metadata"]
					klog.Warningf("%s %s already exists, updating!", obj.GetKind(), obj.GetName())
					_, err = clientDynamic.Resource(gvr).Update(obj, metav1.UpdateOptions{})
				}
			}
		}

		if err != nil {
			return err
		}
	}
	return nil
}

//StatusContainsTypeEqualTo check if u contains a condition type with value typeString
func StatusContainsTypeEqualTo(u *unstructured.Unstructured, typeString string) bool {
	if u != nil {
		if v, ok := u.Object["status"]; ok {
			status := v.(map[string]interface{})
			if v, ok := status["conditions"]; ok {
				conditions := v.([]interface{})
				for _, v := range conditions {
					condition := v.(map[string]interface{})
					if v, ok := condition["type"]; ok {
						if v.(string) == typeString {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

//GetCluster returns the first cluster with a given tag
func GetCluster(tag string, clusters []Cluster) *Cluster {
	for _, cluster := range clusters {
		if tag, ok := cluster.Tags[tag]; ok {
			if tag {
				return &cluster
			}
		}
	}
	return nil
}

//GetClusters returns all clusters with a given tag
func GetClusters(tag string, clusters []Cluster) []*Cluster {
	filteredClusters := make([]*Cluster, 0)
	for i, cluster := range clusters {
		if tag, ok := cluster.Tags[tag]; ok {
			if tag {
				filteredClusters = append(filteredClusters, &clusters[i])
			}
		}
	}
	return filteredClusters
}

// SelectDropDownMenuItem will find a drop down combobox with the initial
// text 'initialComboText', then find a child menu item with text desiredOption.
// The combobox and menu item will be tested to "contain" each string
// rather than for string equality. If an agouti.Selection on the page contains
// the text, it will be "Click()"ed.
func SelectDropDownMenuItem(page *agouti.Page, initialComboText, desiredOption string) error {
	err := ClickSelectionByName(page.AllByClass("bx--list-box__field"), initialComboText)
	if err != nil {
		return err
	}
	//Eventually(page.FindByClass("bx--list-box__menu-item")).Should(BeFound())
	return ClickSelectionByName(page.AllByClass("bx--list-box__menu-item"), desiredOption)
}

// CheckVisibleComboBox all items in a given bare metal cloud connection
func CheckVisibleComboBox(page *agouti.Page, classname string, hosts []Host) error {
	var (
		index          int
		err            error
		multiselection *agouti.MultiSelection
	)

	multiselection = page.AllByClass(classname)
	count, err := page.AllByClass(classname).Count()
	if err != nil {
		fmt.Print("combobox: table should not be empty")
		return err
	}
	for index = 0; index < count; index++ {
		id, err := multiselection.At(index).Attribute("id")
		if err != nil {
			fmt.Print("error combobox items should have id")
			return err
		}
		input, err := multiselection.At(index).FindByClass("bx--checkbox").Attribute("name")
		for i := range hosts {
			if strings.Contains(hosts[i].Name, input) {
				err = multiselection.At(index).Click()
				if err != nil {
					fmt.Print("error when clicking")
					return err
				}
			}
		}
		klog.V(1).Infof("Found checkbox item id: %s, name: %s", id, input)
	}
	return nil
}

// FindMultiSelectionByPlaceholder takes a list of selections
// and clicks the one that has a placeholder text
// equal to the passed in parameter
func FindMultiSelectionByPlaceholder(multiselection *agouti.MultiSelection, placeholder string) *agouti.Selection {
	var err error
	var count, index int
	var text string
	var selection *agouti.Selection

	count, err = multiselection.Count()
	for index = 0; index < count; index++ {
		selection = multiselection.At(index)
		text, err = selection.Attribute("placeholder")
		if err != nil {
			return nil
		}
		klog.V(1).Infof("Found placeholder text: %s", text)
		if strings.Contains(text, placeholder) {
			return selection
		}
	}
	return nil
}

// FindByPlaceholder takes a list of selections
// and clicks the one that has a placeholder text
// equal to the passed in parameter
func FindByPlaceholder(multiselection *agouti.MultiSelection, placeholder string) (*agouti.Selection, error) {
	var err error
	var count, index int
	var text string
	var selection *agouti.Selection

	count, err = multiselection.Count()
	for index = 0; index < count; index++ {
		selection = multiselection.At(index)
		text, err = selection.Attribute("placeholder")
		if err != nil {
			return nil, err
		}
		klog.V(1).Infof("Found placeholder text: %s", text)
		if strings.Contains(text, placeholder) {
			return selection, nil
		}
	}
	return nil, fmt.Errorf("utils: no selection with text \"%s\" could be found within MultiSelect: %+v", placeholder, multiselection)
}

// ClickSelectionByName will iterate through each item in the MultiSelection
// until it comes across an item that contains the `desiredOption` text and
// then send a Click() to it.
func ClickSelectionByName(multiselection *agouti.MultiSelection, desiredOption string) error {
	var err error
	var count, index int
	var text string
	var selection *agouti.Selection

	count, _ = multiselection.Count()
	for index = 0; index < count; index++ {
		selection = multiselection.At(index)
		text, err = selection.Text()
		if err != nil {
			return err
		}
		klog.V(1).Infof("Found selection text: %s", text)
		if strings.Contains(text, desiredOption) {
			return selection.Click()
		}
	}
	return fmt.Errorf("utils: no selection with text \"%s\" could be found within MultiSelect: %+v", desiredOption, multiselection)
}

func HaveServerResources(c Cluster, kubeconfig string, expectedAPIGroups []string) error {
	clientAPIExtension := NewKubeClientAPIExtension(c.MasterURL, kubeconfig, c.KubeContext)
	clientDiscovery := clientAPIExtension.Discovery()
	for _, apiGroup := range expectedAPIGroups {
		klog.V(1).Infof("Check if %s exists", apiGroup)
		_, err := clientDiscovery.ServerResourcesForGroupVersion(apiGroup)
		if err != nil {
			klog.V(1).Infof("Error while retrieving server resource %s: %s", apiGroup, err.Error())
			return err
		}
	}
	return nil
}

func HaveCRDs(c Cluster, kubeconfig string, expectedCRDs []string) error {
	clientAPIExtension := NewKubeClientAPIExtension(c.MasterURL, kubeconfig, c.KubeContext)
	clientAPIExtensionV1beta1 := clientAPIExtension.ApiextensionsV1beta1()
	for _, crd := range expectedCRDs {
		klog.V(1).Infof("Check if %s exists", crd)
		_, err := clientAPIExtensionV1beta1.CustomResourceDefinitions().Get(crd, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving crd %s: %s", crd, err.Error())
			return err
		}
	}
	return nil
}

func HaveDeploymentsInNamespace(c Cluster, kubeconfig string, namespace string, expectedDeploymentNames []string) error {

	client := NewKubeClient(c.MasterURL, kubeconfig, c.KubeContext)
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	klog.V(1).Infof("Server version info: %v", versionInfo)

	deployments := client.AppsV1().Deployments(namespace)

	for _, deploymentName := range expectedDeploymentNames {
		klog.V(1).Infof("Check if deployment %s exists", deploymentName)
		deployment, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving deployment %s: %s", deploymentName, err.Error())
			return err
		}
		if deployment.Status.Replicas != deployment.Status.ReadyReplicas {
			err = fmt.Errorf("Expect %d but got %d Ready replicas", deployment.Status.Replicas, deployment.Status.ReadyReplicas)
			klog.Errorln(err)
			return err
		}
		for _, condition := range deployment.Status.Conditions {
			if condition.Reason == "MinimumReplicasAvailable" {
				if condition.Status != corev1.ConditionTrue {
					err = fmt.Errorf("Expect %s but got %s", condition.Status, corev1.ConditionTrue)
					klog.Errorln(err)
					return err
				}
			}
		}
	}

	return nil
}

func HaveStatefulSetsInNamespace(c Cluster, kubeconfig string, namespace string, expectedStatefulSetNames []string) error {
	client := NewKubeClient(c.MasterURL, kubeconfig, c.KubeContext)
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	klog.V(1).Infof("Server version info: %v", versionInfo)

	statefulsets := client.AppsV1().StatefulSets(namespace)

	for _, statefulsetName := range expectedStatefulSetNames {
		klog.V(1).Infof("Check if statefulset %s exists", statefulsetName)
		statefulset, err := statefulsets.Get(statefulsetName, metav1.GetOptions{})
		if err != nil {
			klog.V(1).Infof("Error while retrieving statefulset %s: %s", statefulsetName, err.Error())
			return err
		}
		if statefulset.Status.Replicas != statefulset.Status.ReadyReplicas {
			err = fmt.Errorf("Expect %d but got %d Ready replicas", statefulset.Status.Replicas, statefulset.Status.ReadyReplicas)
			klog.Errorln(err)
			return err
		}
	}

	return nil
}

func GetKubeVersion(client *rest.RESTClient) version.Info {
	kubeVersion := version.Info{}

	versionBody, err := client.Get().AbsPath("/version").Do().Raw()
	if err != nil {
		log.Error(err, "fail to GET /version")
		return version.Info{}
	}

	err = json.Unmarshal(versionBody, &kubeVersion)
	if err != nil {
		log.Error(fmt.Errorf("fail to Unmarshal, got '%s': %v", string(versionBody), err), "")
		return version.Info{}
	}

	return kubeVersion
}

func IsOpenshift(client *rest.RESTClient) bool {
	//check whether the cluster is openshift or not for openshift version 3.11 and before
	_, err := client.Get().AbsPath("/version/openshift").Do().Raw()
	if err == nil {
		klog.V(5).Info("Found openshift version from /version/openshift")
		return true
	}

	//check whether the cluster is openshift or not for openshift version 4.1
	_, err = client.Get().AbsPath("/apis/config.openshift.io/v1/clusterversions").Do().Raw()
	if err == nil {
		klog.V(5).Info("Found openshift version from /apis/config.openshift.io/v1/clusterversions")
		return true
	}

	klog.V(5).Infof("fail to GET openshift version, assuming not OpenShift: %s", err.Error())
	return false
}
