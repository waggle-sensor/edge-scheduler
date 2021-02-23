package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/sagecontinuum/ses/pkg/knowledgebase"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"k8s.io/client-go/util/homedir"
)

var (
	rabbitmqManagementURI      string
	rabbitmqManagementUsername string
	rabbitmqManagementPassword string
	kubeconfig                 string
	registry                   string
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {

	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&registry, "registry", "waggle/", "Path to ECR registry")
	flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", "http://10.31.81.10:15672"), "rabbitmq management uri")
	flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "guest"), "rabbitmq management username")
	flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "guest"), "rabbitmq management password")
	flag.Parse()

	k3sClient, err := nodescheduler.GetK3SClient(kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	rmqClient, err := rabbithole.NewClient(rabbitmqManagementURI, rabbitmqManagementUsername, rabbitmqManagementPassword)
	if err != nil {
		log.Fatal(err)
	}

	rm, err := nodescheduler.NewK3SResourceManager("default", registry, k3sClient, rmqClient)
	if err != nil {
		log.Fatal(err)
	}

	kb, err := knowledgebase.NewKnowledgebase()
	if err != nil {
		log.Fatal(err)
	}

	gm, err := nodescheduler.NewGoalManager("http://localhost:9770", "vagrant")
	if err != nil {
		log.Fatal(err)
	}

	ns, err := nodescheduler.NewNodeScheduler(rm, kb, gm)
	if err != nil {
		log.Fatal(err)
	}

	ns.Run()
}
