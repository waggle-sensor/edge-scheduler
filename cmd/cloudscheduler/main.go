package main

import (
	"flag"
	"log"

	"github.com/sagecontinuum/ses/pkg/cloudscheduler"
	"github.com/sagecontinuum/ses/pkg/logger"
)

var (
	sesName string
	// rabbitmqManagementURI      string
	// rabbitmqManagementUsername string
	// rabbitmqManagementPassword string
	registry string
	port     int
	dataDir  string
)

func main() {
	flag.StringVar(&sesName, "sesname", "ses", "Name of cloud scheduler")
	// flag.StringVar(&registry, "registry", "waggle/", "Path to ECR registry")
	flag.IntVar(&port, "port", 9770, "Port to listen")
	flag.StringVar(&dataDir, "data-dir", "data", "Path to meta directory")
	flag.Parse()

	// TODO: a RMQ client for goal manager will be needed
	// flag.StringVar(&rabbitmqManagementURI, "rabbitmq-management-uri", getenv("RABBITMQ_MANAGEMENT_URI", "http://rabbitmq:15672"), "rabbitmq management uri")
	// flag.StringVar(&rabbitmqManagementUsername, "rabbitmq-management-username", getenv("RABBITMQ_MANAGEMENT_USERNAME", "guest"), "rabbitmq management username")
	// flag.StringVar(&rabbitmqManagementPassword, "rabbitmq-management-password", getenv("RABBITMQ_MANAGEMENT_PASSWORD", "guest"), "rabbitmq management password")

	logger.Info.Printf("Cloud scheduler (%s) starts...", sesName)

	// rmqManagement, err := nodescheduler.NewRMQManagement(rabbitmqManagementURI, rabbitmqManagementUsername, rabbitmqManagementPassword)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	mh, err := cloudscheduler.NewMetaHandler(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	cs, err := cloudscheduler.NewCloudScheduler(mh)
	if err != nil {
		log.Fatal(err)
	}
	cs.Run(sesName, port)
}
