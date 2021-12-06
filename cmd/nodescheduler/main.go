package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/sagecontinuum/ses/pkg/knowledgebase"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
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
		simulate                   bool
		noRabbitMQ                 bool
		rabbitmqURI                string
		rabbitmqUsername           string
		rabbitmqPassword           string
		rabbitmqManagementURI      string
		rabbitmqManagementUsername string
		rabbitmqManagementPassword string
		kubeconfig                 string
		registry                   string
		cloudschedulerURI          string
		nodeID                     string
		incluster                  bool
	)
	flag.BoolVar(&simulate, "simulate", false, "Simulate the scheduler")
	flag.StringVar(&nodeID, "nodeid", getenv("WAGGLE_NODE_ID", "000000000001"), "node ID")
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.BoolVar(&incluster, "in-cluster", false, "A flag indicating a k3s service account is used")
	flag.StringVar(&registry, "registry", "waggle/", "Path to ECR registry")
	flag.BoolVar(&noRabbitMQ, "no-rabbitmq", false, "No RabbitMQ to talk to the cloud scheduler")
	flag.StringVar(&rabbitmqURI, "rabbitmq-uri", getenv("RABBITMQ_URI", "rabbitmq:5672"), "RabbitMQ management uri")
	flag.StringVar(&rabbitmqUsername, "rabbitmq-username", getenv("RABBITMQ_USERNAME", "guest"), "RabbitMQ management username")
	flag.StringVar(&rabbitmqPassword, "rabbitmq-password", getenv("RABBITMQ_PASSWORD", "guest"), "RabbitMQ management password")
	flag.StringVar(&cloudschedulerURI, "cloudscheduler-uri", "http://localhost:9770", "cloudscheduler URI")
	flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", "http://rabbitmq:15672"), "rabbitmq management uri")
	flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "guest"), "rabbitmq management username")
	flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "guest"), "rabbitmq management password")
	flag.Parse()
	logger.Info.Print("Nodescheduler starts...")
	logger.Debug.Print("Creating RMQ management instance...")
	rmqManagement, err := nodescheduler.NewRMQManagement(rabbitmqManagementURI, rabbitmqManagementUsername, rabbitmqManagementPassword, simulate)
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug.Print("Creating resource manager instance...")
	rm, err := nodescheduler.NewK3SResourceManager(registry, incluster, kubeconfig, rmqManagement, simulate)
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug.Print("Creating knowledgebase instance...")
	u, err := url.Parse(rabbitmqManagementURI)
	if err != nil {
		log.Fatal(err)
	}
	kb, err := knowledgebase.NewKnowledgebase(u.Hostname())
	if err != nil {
		log.Fatal(err)
	}
	logger.Info.Printf("My Node ID is %s", nodeID)
	logger.Debug.Print("Creating goal manager instance...")
	gm, err := nodescheduler.NewNodeGoalManager(cloudschedulerURI, nodeID, simulate)
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug.Print("Creating API server instance...")
	api, err := nodescheduler.NewAPIServer()
	if err != nil {
		log.Fatal(err)
	}
	ns, err := nodescheduler.NewNodeScheduler(rm, kb, gm, api, simulate)
	if err != nil {
		log.Fatal(err)
	}
	if !noRabbitMQ {
		logger.Info.Printf("Using RabbitMQ at %s with user %s", rabbitmqURI, rabbitmqUsername)
		rmqHandler := interfacing.NewRabbitMQHandler(rabbitmqURI, rabbitmqUsername, rabbitmqPassword)
		ns.GoalManager.SetRMQHandler(rmqHandler)
	}
	ns.Configure()
	ns.Run()
}
