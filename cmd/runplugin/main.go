package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

func main() {
	var (
		kubeconfig                 string
		rabbitmqManagementURI      string
		rabbitmqManagementUsername string
		rabbitmqManagementPassword string
	)

	if val, ok := os.LookupEnv("KUBECONFIG"); ok {
		flag.StringVar(&kubeconfig, "kubeconfig", val, "absolute path to the kubeconfig file")
	} else if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", "http://localhost:15672"), "rabbitmq management uri")
	flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "guest"), "rabbitmq management username")
	flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "guest"), "rabbitmq management password")
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
	if err := sch.RunPlugin(args[0], args[1:]...); err != nil {
		log.Fatal(err)
	}
}
