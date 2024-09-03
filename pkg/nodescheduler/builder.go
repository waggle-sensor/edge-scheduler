package nodescheduler

import (
	"strings"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler/policy"
)

type NodeSchedulerConfig struct {
	Name             string `json:"nodename" yaml:"nodeName"`
	Version          string
	NoRabbitMQ       bool   `json:"no_rabbitmq" yaml:"noRabbitMQ"`
	RabbitmqURI      string `json:"rabbitmq_uri" yaml:"rabbimqURI"`
	RabbitmqUsername string `json:"rabbitmq_username" yaml:"rabbitMQUsername"`
	RabbitmqPassword string `json:"rabbitmq_password" yaml:"rabbitMQPassword"`
	Kubeconfig       string `json:"kubeconfig" yaml:"kubeConfig"`
	InCluster        bool   `json:"in_cluster" yaml:"inCluster"`
	RuleCheckerURI   string `json:"rulechecker_uri" yaml:"ruleCheckerURI"`
	ScoreboardURI    string `json:"scoreboard_uri" yaml:"scoreboardURI"`
	Simulate         bool   `json:"simulate" yaml:"simulate"`
	GoalStreamURL    string `json:"goalstream_URI" yaml:"goalStreamURL"`
	SchedulingPolicy string `json:"policy" yaml:"policy"`
	Debug            bool   `json:"debug" yaml:"debug"`
}

type NodeSchedulerBuilder struct {
	nodeScheduler *NodeScheduler
}

func NewNodeSchedulerBuilder(config *NodeSchedulerConfig) *NodeSchedulerBuilder {
	return &NodeSchedulerBuilder{
		nodeScheduler: &NodeScheduler{
			Version:                     config.Version,
			NodeID:                      strings.ToLower(config.Name),
			Config:                      config,
			SchedulingPolicy:            policy.GetSchedulingPolicyByName(config.SchedulingPolicy),
			chanContextEventToScheduler: make(chan datatype.EventPluginContext, maxChannelBuffer),
			chanFromResourceManager:     make(chan datatype.Event, maxChannelBuffer),
			chanFromCloudScheduler:      make(chan datatype.Event, maxChannelBuffer),
			chanNeedScheduling:          make(chan datatype.Event, maxChannelBuffer),
		},
	}
}

func (nsb *NodeSchedulerBuilder) AddGoalManager(appID string) *NodeSchedulerBuilder {
	nsb.nodeScheduler.GoalManager = &NodeGoalManager{
		ScienceGoals:  make(map[string]datatype.ScienceGoal),
		LoadedPlugins: make(map[PluginIndex]*datatype.PluginRuntime),
	}
	return nsb
}

func (nsb *NodeSchedulerBuilder) AddResourceManager() *NodeSchedulerBuilder {
	nsb.nodeScheduler.ResourceManager = &ResourceManager{
		Namespace:     "ses",
		Clientset:     nil,
		MetricsClient: nil,
		Simulate:      nsb.nodeScheduler.Config.Simulate,
		Notifier:      interfacing.NewNotifier(),
		runner:        "nodescheduler",
	}
	nsb.nodeScheduler.ResourceManager.Notifier.Subscribe(nsb.nodeScheduler.chanFromResourceManager)
	return nsb
}

func (nsb *NodeSchedulerBuilder) AddKnowledgebase() *NodeSchedulerBuilder {
	nsb.nodeScheduler.Knowledgebase = &KnowledgeBase{
		nodeID:         nsb.nodeScheduler.Config.Name,
		rules:          make(map[string][]datatype.ScienceRule),
		measures:       map[string]interface{}{},
		ruleCheckerURI: nsb.nodeScheduler.Config.RuleCheckerURI,
	}
	return nsb
}

func (nsb *NodeSchedulerBuilder) AddAPIServer() *NodeSchedulerBuilder {
	nsb.nodeScheduler.APIServer = &APIServer{
		version:       nsb.nodeScheduler.Config.Version,
		nodeScheduler: nsb.nodeScheduler,
		port:          8080,
	}
	return nsb
}

func (nsb *NodeSchedulerBuilder) AddLoggerToBeehive(appID string) *NodeSchedulerBuilder {
	nsb.nodeScheduler.LogToBeehive = interfacing.NewRabbitMQHandler(
		nsb.nodeScheduler.Config.RabbitmqURI,
		nsb.nodeScheduler.Config.RabbitmqUsername,
		nsb.nodeScheduler.Config.RabbitmqPassword,
		"",
		appID)
	return nsb
}

func (nsb *NodeSchedulerBuilder) AddConnToScoreboard() *NodeSchedulerBuilder {
	nsb.nodeScheduler.ToScoreboard = interfacing.NewRedisClient(nsb.nodeScheduler.Config.ScoreboardURI)
	return nsb
}

func (nsb *NodeSchedulerBuilder) Build() *NodeScheduler {
	return nsb.nodeScheduler
}
