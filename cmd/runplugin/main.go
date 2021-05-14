package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/sagecontinuum/ses/pkg/runplugin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

const rancherKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"

func detectDefaultKubeconfig() string {
	if _, err := os.ReadFile(rancherKubeconfigPath); err == nil {
		return rancherKubeconfigPath
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

func detectDefaultRabbitmqURI() string {
	if b, err := exec.Command("kubectl", "get", "svc", "wes-rabbitmq", "-o", "jsonpath=http://{.spec.clusterIP}:15672").CombinedOutput(); err == nil {
		return string(b)
	}
	return "http://localhost:15672"
}

func main() {
	var (
		privileged                 bool
		node                       string
		kubeconfig                 string
		rabbitmqManagementURI      string
		rabbitmqManagementUsername string
		rabbitmqManagementPassword string
	)

	flag.BoolVar(&privileged, "privileged", false, "run as privileged plugin")
	flag.StringVar(&node, "node", "", "run plugin on node")
	flag.StringVar(&kubeconfig, "kubeconfig", getenv("KUBECONFIG", detectDefaultKubeconfig()), "path to the kubeconfig file")
	flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", detectDefaultRabbitmqURI()), "rabbitmq management uri")
	flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "admin"), "rabbitmq management username")
	flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "admin"), "rabbitmq management password")
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		fmt.Printf("%s plugin-image [plugin-args]", os.Args[0])
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	rmqclient, err := rabbithole.NewClient(rabbitmqManagementURI, rabbitmqManagementUsername, rabbitmqManagementPassword)
	if err != nil {
		log.Fatal(err)
	}

	sch := &runplugin.Scheduler{
		KubernetesClientset: clientset,
		RabbitMQClient:      rmqclient,
	}

	args := flag.Args()

	spec := &runplugin.Spec{
		Privileged: privileged,
		Node:       node,
		Image:      args[0],
		Args:       args[1:],
	}

	if err := sch.RunPlugin(spec); err != nil {
		log.Fatal(err)
	}
}
