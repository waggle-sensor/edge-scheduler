package main

import (
	"flag"
	"log"
	"os"

	"github.com/sagecontinuum/ses/pkg/cloudscheduler"
	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/sagecontinuum/ses/pkg/logger"
)

var (
	sesName          string
	noRabbitMQ       bool
	rabbitmqURI      string
	rabbitmqUsername string
	rabbitmqPassword string
	registry         string
	port             int
	dataDir          string
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	flag.StringVar(&sesName, "sesname", "ses", "Name of cloud scheduler")
	// flag.StringVar(&registry, "registry", "waggle/", "Path to ECR registry")
	flag.IntVar(&port, "port", 9770, "Port to listen")
	flag.StringVar(&dataDir, "data-dir", "data", "Path to meta directory")

	// TODO: a RMQ client for goal manager will be needed
	flag.BoolVar(&noRabbitMQ, "no-rabbitmq", false, "No RabbitMQ to talk to edge schedulers")
	flag.StringVar(&rabbitmqURI, "rabbitmq-uri", getenv("RABBITMQ_URI", "rabbitmq:5672"), "RabbitMQ management uri")
	flag.StringVar(&rabbitmqUsername, "rabbitmq-username", getenv("RABBITMQ_USERNAME", "guest"), "RabbitMQ management username")
	flag.StringVar(&rabbitmqPassword, "rabbitmq-password", getenv("RABBITMQ_PASSWORD", "guest"), "RabbitMQ management password")
	flag.Parse()

	logger.Info.Printf("Cloud scheduler (%s) starts...", sesName)

	mh, err := cloudscheduler.NewMetaHandler(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	cs, err := cloudscheduler.NewCloudScheduler(mh)
	if err != nil {
		log.Fatal(err)
	}

	if !noRabbitMQ {
		logger.Info.Printf("Using RabbitMQ at %s with user %s", rabbitmqURI, rabbitmqUsername)
		rmqHandler := interfacing.NewRabbitMQHandler(rabbitmqURI, rabbitmqUsername, rabbitmqPassword, "")
		cs.GoalManager.SetRMQHandler(rmqHandler)
	}
	cs.Run(sesName, port)
}
