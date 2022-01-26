package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/sagecontinuum/ses/pkg/interfacing"
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
		simulate          bool
		noRabbitMQ        bool
		rabbitmqURI       string
		rabbitmqUsername  string
		rabbitmqPassword  string
		kubeconfig        string
		registry          string
		cloudschedulerURI string
		nodeID            string
		incluster         bool
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
	flag.Parse()
	logger.Info.Println("Nodescheduler starts...")
	logger.Info.Printf("My Node ID is %s", nodeID)
	var ns *nodescheduler.NodeScheduler
	if simulate {
		logger.Debug.Println("Creating scheduler for simulation...")
		ns = nodescheduler.NewFakeNodeSchedulerBuilder(nodeID).
			AddGoalManager().
			AddResourceManager().
			AddAPIServer().
			Build()
	} else {
		logger.Debug.Println("Creating scheduler for real...")
		ns = nodescheduler.NewRealNodeSchedulerBuilder(nodeID).
			AddGoalManager(cloudschedulerURI).
			AddResourceManager(registry, incluster, kubeconfig).
			AddAPIServer().
			Build()
	}
	if !noRabbitMQ {
		logger.Info.Printf("Using RabbitMQ at %s with user %s", rabbitmqURI, rabbitmqUsername)
		rmqHandler := interfacing.NewRabbitMQHandler(rabbitmqURI, rabbitmqUsername, rabbitmqPassword)
		ns.GoalManager.SetRMQHandler(rmqHandler)
	}
	ns.Configure()
	ns.Run()
}
