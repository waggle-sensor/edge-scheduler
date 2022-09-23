package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"gopkg.in/yaml.v2"
)

var Version string

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	var config cloudscheduler.CloudSchedulerConfig
	var configPath string
	config.Version = Version
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&config.Name, "name", "cloudscheduler-sage", "Name of cloud scheduler")
	// TODO: Add ECRURL to query meta information for plugins when validating a job
	// flag.StringVar(&ECRURI, "ECRURL", "SOMEWHERE", "Path to ECR URL")
	flag.IntVar(&config.Port, "port", 9770, "Port to listen")
	flag.StringVar(&config.DataDir, "data-dir", "data", "Path to meta directory")
	// TODO: a RMQ client for goal manager will be needed
	flag.BoolVar(&config.NoRabbitMQ, "no-rabbitmq", false, "No RabbitMQ to talk to edge schedulers")
	flag.StringVar(&config.RabbitmqURI, "rabbitmq-uri", getenv("RABBITMQ_URI", "rabbitmq:5672"), "RabbitMQ management uri")
	flag.StringVar(&config.RabbitmqUsername, "rabbitmq-username", getenv("RABBITMQ_USERNAME", "guest"), "RabbitMQ management username")
	flag.StringVar(&config.RabbitmqPassword, "rabbitmq-password", getenv("RABBITMQ_PASSWORD", "guest"), "RabbitMQ management password")
	flag.BoolVar(&config.PushNotification, "push-notification", true, "Enable HTTP push notification for science goals")
	flag.StringVar(&config.AuthServerURL, "auth-server-url", getenv("AUTH_URL", ""), "Authentication server URL")
	flag.StringVar(&config.AuthPassword, "auth-password", getenv("AUTH_PASSWORD", ""), "Password to query in authentication server")
	flag.Parse()
	logger.Info.Printf("Cloud scheduler (%s) starts...", config.Name)
	if configPath != "" {
		logger.Info.Printf("Config file (%s) provided. Loading configs...", configPath)
		blob, err := ioutil.ReadFile(configPath)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(blob, &config)
		if err != nil {
			panic(err)
		}
	}
	cs := cloudscheduler.NewCloudSchedulerBuilder(&config).
		AddGoalManager().
		AddAPIServer().
		Build()

	err := cs.Configure()
	if err != nil {
		panic(err)
	}
	cs.Run()
}
