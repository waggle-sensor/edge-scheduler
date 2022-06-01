package main

import (
	"flag"
	"os"

	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

var Version string

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	var (
		name             string
		noRabbitMQ       bool
		rabbitmqURI      string
		rabbitmqUsername string
		rabbitmqPassword string
		// ECRURL         string
		port    int
		dataDir string
	)
	flag.StringVar(&name, "name", "ses-sage", "Name of cloud scheduler")
	// TODO: Add ECRURL to query meta information for plugins when validating a job
	// flag.StringVar(&ECRURL, "ECRURL", "SOMEWHERE", "Path to ECR URL")
	flag.IntVar(&port, "port", 9770, "Port to listen")
	flag.StringVar(&dataDir, "data-dir", "data", "Path to meta directory")
	// TODO: a RMQ client for goal manager will be needed
	flag.BoolVar(&noRabbitMQ, "no-rabbitmq", false, "No RabbitMQ to talk to edge schedulers")
	flag.StringVar(&rabbitmqURI, "rabbitmq-uri", getenv("RABBITMQ_URI", "rabbitmq:5672"), "RabbitMQ management uri")
	flag.StringVar(&rabbitmqUsername, "rabbitmq-username", getenv("RABBITMQ_USERNAME", "guest"), "RabbitMQ management username")
	flag.StringVar(&rabbitmqPassword, "rabbitmq-password", getenv("RABBITMQ_PASSWORD", "guest"), "RabbitMQ management password")
	flag.Parse()

	logger.Info.Printf("Cloud scheduler (%s) starts...", name)

	cs := cloudscheduler.NewRealCloudSchedulerBuilder(name, Version).
		AddGoalManager().
		AddAPIServer(port).
		Build()

	if !noRabbitMQ {
		logger.Info.Printf("Using RabbitMQ at %s with user %s", rabbitmqURI, rabbitmqUsername)
		rmqHandler := interfacing.NewRabbitMQHandler(rabbitmqURI, rabbitmqUsername, rabbitmqPassword, "")
		cs.GoalManager.SetRMQHandler(rmqHandler)
	}
	err := cs.Validator.LoadDatabase(dataDir)
	if err != nil {
		panic(err)
	}
	cs.Run()
}
