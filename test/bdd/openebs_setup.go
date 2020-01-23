package bdd

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	chaosClient "github.com/litmuschaos/chaos-operator/pkg/client/clientset/versioned/typed/litmuschaos/v1alpha1"
	restclient "k8s.io/client-go/rest"
)

var (
	kubeconfig string
	config     *restclient.Config
	client     *kubernetes.Clientset
	clientSet  *chaosClient.LitmuschaosV1alpha1Client
)

func TestChaos(t *testing.T) {

	RegisterFailHandler(Fail)
	RunSpecs(t, "BDD test")
}

var _ = BeforeSuite(func() {

	var err error
	kubeconfig = os.Getenv("HOME") + "/.kube/config"
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		Expect(err).To(BeNil(), "failed to get config")
	}

	client, err = kubernetes.NewForConfig(config)

	if err != nil {
		Expect(err).To(BeNil(), "failed to get client")
	}

	clientSet, err = chaosClient.NewForConfig(config)

	if err != nil {
		Expect(err).To(BeNil(), "failed to get clientSet")
	}

	err = v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		fmt.Println(err)
	}

	//Creating crds
	By("Creating chaosengine crd")
	err = exec.Command("kubectl", "apply", "-f", "https://raw.githubusercontent.com/litmuschaos/chaos-operator/master/deploy/chaos_crds.yaml").Run()
	Expect(err).To(BeNil(), "failed to create crds")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("crds installed successfully")

	//Creating rbacs
	By("Creating chaosengine rbac")
	err = exec.Command("kubectl", "apply", "-f", "https://raw.githubusercontent.com/litmuschaos/chaos-operator/master/deploy/rbac.yaml").Run()
	Expect(err).To(BeNil(), "failed to create rbac")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("rbac installed sucessfully")

	//Creating Chaos-Operator
	By("creating operator")
	err = exec.Command("kubectl", "create", "-f", "https://raw.githubusercontent.com/litmuschaos/chaos-operator/master/deploy/operator.yaml").Run()
	Expect(err).To(BeNil(), "failed to create chaos-operator")
	if err != nil {
		fmt.Println(err)
	}

	operator, err := client.AppsV1().Deployments("litmus").Get("chaos-operator", metav1.GetOptions{})
	for operator.Status.UnavailableReplicas != 0 {
		fmt.Printf("Unavaliable Count: %v", operator.Status.UnavailableReplicas)
		operator, _ = client.AppsV1().Deployments("litmus").Get("chaos-operator", metav1.GetOptions{})
	}

	fmt.Println("Chaos-operator created successfully")

	// Installing OpenEBS

	By("Installing OpenEBS")
	err = exec.Command("kubectl", "apply", "-f", "https://openebs.github.io/charts/openebs-operator-1.2.0.yaml").Run()
	Expect(err).To(BeNil(), "failed to install openebs")
	if err != nil {
		fmt.Println(err)
	}

	// Checking status of OpenEBS Pods
	f := 0

	for i := 0; i < 60; i++ {
		f = 0
		totalPods, _ := client.CoreV1().Pods("openebs").List(metav1.ListOptions{})

		for _, v := range totalPods.Items {
			if v.Status.Phase != "Running" {
				f = 1
				break
			}
		}
		if f == 0 {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if f == 0 {
		fmt.Printf("All openebs pods are in running state")
	} else {
		fmt.Printf("All openebs pods are not in running state")
	}

	fmt.Println("Openebs Installed successfully")

	//Install OpenEBS Chaos Charts

	By("Installing OpenEBS Charts")
	err = exec.Command("wget", "https://openebs.github.io/charts/openebs-operator-1.2.0.yaml").Run()
	Expect(err).To(BeNil(), "failed to install OpenEBS Charts")
	if err != nil {
		fmt.Println(err)
	}

	//Changing the required field

	By("Changing the required field")
	err = exec.Command("sed", "-i", "157 s/false/true/", "openebs-operator-1.2.0.yaml").Run()
	Expect(err).To(BeNil(), "failed to make changes in OpenEBS Charts")
	if err != nil {
		fmt.Println(err)
	}

	//Creating Pools
	By("Creating OpenEBs Pool")
	err = exec.Command("kubectl", "apply", "-f", "openebs-operator-1.2.0.yaml").Run()
	Expect(err).To(BeNil(), "failed to Create OpenEBS pool")
	if err != nil {
		fmt.Println(err)
	}

})