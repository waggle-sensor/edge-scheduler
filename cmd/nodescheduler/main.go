package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/sagecontinuum/ses/pkg/knowledgebase"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"k8s.io/client-go/util/homedir"
)

var (
	rabbitmqManagementURI      string
	rabbitmqManagementUsername string
	rabbitmqManagementPassword string
	kubeconfig                 string
	registry                   string
	cloudschedulerURI          string
	nodeID                     string
	incluster                  bool
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	flag.StringVar(&nodeID, "nodeid", getenv("WAGGLE_NODE_ID", "000000000001"), "node ID")
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.BoolVar(&incluster, "in-cluster", false, "a flag indicating a k3s service account is used")
	flag.StringVar(&registry, "registry", "waggle/", "Path to ECR registry")
	flag.StringVar(&cloudschedulerURI, "cloudscheduler-uri", "http://localhost:9770", "cloudscheduler URI")
	flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", "http://rabbitmq:15672"), "rabbitmq management uri")
	flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "guest"), "rabbitmq management username")
	flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "guest"), "rabbitmq management password")
	flag.Parse()

	logger.Info.Print("Nodescheduler starts...")

	k3sClient, err := nodescheduler.GetK3SClient(incluster, kubeconfig)
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

	u, err := url.Parse(rabbitmqManagementURI)
	if err != nil {
		log.Fatal(err)
	}
	kb, err := knowledgebase.NewKnowledgebase(u.Hostname())
	if err != nil {
		log.Fatal(err)
	}

	gm, err := nodescheduler.NewGoalManager(cloudschedulerURI, "vagrant")
	if err != nil {
		log.Fatal(err)
	}

	ns, err := nodescheduler.NewNodeScheduler(rm, kb, gm)
	if err != nil {
		log.Fatal(err)
	}

	ns.Run()
}
