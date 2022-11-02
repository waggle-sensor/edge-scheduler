package main

import (
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/homedir"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

var Version = "0.0.0"

func main() {
	var config nodescheduler.NodeSchedulerConfig
	var configPath string
	config.Version = Version
	flag.BoolVar(&config.Debug, "debug", false, "flag to debug")
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&config.Simulate, "simulate", false, "Simulate the scheduler")
	flag.StringVar(&config.Name, "nodename", getenv("WAGGLE_NODE_VSN", "W000"), "node name (VSN)")
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&config.Kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&config.Kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.BoolVar(&config.InCluster, "in-cluster", false, "A flag indicating a k3s service account is used")
	flag.BoolVar(&config.NoRabbitMQ, "no-rabbitmq", false, "No RabbitMQ to talk to the cloud scheduler")
	flag.StringVar(&config.RabbitmqURI, "rabbitmq-uri", getenv("RABBITMQ_URI", "wes-rabbitmq:5672"), "RabbitMQ management uri")
	flag.StringVar(&config.RabbitmqUsername, "rabbitmq-username", getenv("RABBITMQ_USERNAME", "service"), "RabbitMQ management username")
	flag.StringVar(&config.RabbitmqPassword, "rabbitmq-password", getenv("RABBITMQ_PASSWORD", "service"), "RabbitMQ management password")
	flag.StringVar(&config.GoalStreamURL, "goalstream-url", "", "URL to receive goal stream")
	flag.StringVar(&config.RuleCheckerURI, "rulechecker-uri", "http://wes-sciencerule-checker:5000", "rulechecker URI")
	flag.StringVar(&config.SchedulingPolicy, "policy", "default", "Name of the scheduling policy")
	flag.Parse()
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
	if !config.Debug {
		logger.Debug.SetOutput(io.Discard)
	}
	logger.Info.Printf("Node scheduler (%q) starts...", config.Name)
	logger.Debug.Print("Creating node scheduler...")
	appID := getenv("WAGGLE_APP_ID", "")
	ns := nodescheduler.NewNodeSchedulerBuilder(&config).
		AddGoalManager(appID).
		AddKnowledgebase().
		AddResourceManager().
		AddAPIServer().
		AddLoggerToBeehive(appID).
		Build()
	err := ns.Configure()
	if err != nil {
		panic(err)
	}
	ns.Run()
}
