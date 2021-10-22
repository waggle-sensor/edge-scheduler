package main

import (
	"flag"
	"log"
	"os"

	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"github.com/sagecontinuum/ses/pkg/simplescheduler"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	var (
		registry                   string
		rabbitmqManagementURI      string
		rabbitmqManagementUsername string
		rabbitmqManagementPassword string
	)
	flag.StringVar(&registry, "registry", "waggle/", "Path to ECR registry")
	flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", "http://wes-rabbitmq:15672"), "rabbitmq management uri")
	flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "admin"), "rabbitmq management username")
	flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "admin"), "rabbitmq management password")
	// Assume this to be running inside a Kubernetes cluster
	k3sClient, err := nodescheduler.GetK3SClient(false, "/Users/yongho.kim/tmp/maingate_kubeconfig")
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
	err = scheduler.Configure()
	if err != nil {
		log.Fatal(err)
	}
	scheduler.Run()
}
