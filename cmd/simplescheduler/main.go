package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"github.com/sagecontinuum/ses/pkg/simplescheduler"
	"k8s.io/client-go/util/homedir"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func detectDefaultKubeconfig() string {
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

func main() {
	var (
		kubeconfig string
		registry   string
	)
	flag.StringVar(&kubeconfig, "kubeconfig", getenv("KUBECONFIG", detectDefaultKubeconfig()), "path to the kubeconfig file")
	flag.StringVar(&registry, "registry", "", "Path to ECR registry")
	// Assume this to be running inside a Kubernetes cluster
	k3sClient, err := nodescheduler.GetK3SClient(false, kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	rmqManagement, err := nodescheduler.NewRMQManagement(rabbitmqManagementURI, rabbitmqManagementUsername, rabbitmqManagementPassword)
	if err != nil {
		log.Fatal(err)
	}

	rm, err := nodescheduler.NewK3SResourceManager("default", registry, k3sClient, rmqManagement)
	if err != nil {
		log.Fatal(err)
	}

	scheduler := simplescheduler.NewSimpleScheduler(rm)
	// err = scheduler.Configure()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	scheduler.Run()
}
