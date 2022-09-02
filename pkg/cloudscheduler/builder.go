package cloudscheduler

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type CloudSchedulerConfig struct {
	Name             string `json:"name" yaml:"name"`
	Version          string
	NoRabbitMQ       bool   `json:"no_rabbitmq" yaml:"noRabbitMQ"`
	RabbitmqURI      string `json:"rabbitmq_uri" yaml:"rabbimqURI"`
	RabbitmqUsername string `json:"rabbitmq_username" yaml:"rabbitMQUsername"`
	RabbitmqPassword string `json:"rabbitmq_password" yaml:"rabbitMQPassword"`
	ECRURI           string `json:"ecr_uri" yaml:"ecrURI"`
	Port             int    `json:"port" yaml:"port"`
	DataDir          string `json:"data_dir,omitempty" yaml:"dataDir,omitempty"`
	PushNotification bool   `json:"push_notification" yaml:"PushNotification"`
}

type CloudSchedulerBuilder struct {
	cloudScheduler *CloudScheduler
}

func NewCloudSchedulerBuilder(config *CloudSchedulerConfig) *CloudSchedulerBuilder {
	return &CloudSchedulerBuilder{
		cloudScheduler: &CloudScheduler{
			Name:                config.Name,
			Version:             config.Version,
			Config:              config,
			Validator:           NewJobValidator(config.DataDir),
			chanFromGoalManager: make(chan datatype.Event, maxChannelBuffer),
		},
	}
}

func (csb *CloudSchedulerBuilder) AddGoalManager() *CloudSchedulerBuilder {
	csb.cloudScheduler.GoalManager = &CloudGoalManager{
		scienceGoals: make(map[string]*datatype.ScienceGoal),
		Notifier:     interfacing.NewNotifier(),
		dataPath:     csb.cloudScheduler.Config.DataDir,
	}
	if !csb.cloudScheduler.Config.NoRabbitMQ {
		logger.Info.Printf(
			"Using RabbitMQ at %s with user %s",
			csb.cloudScheduler.Config.RabbitmqURI,
			csb.cloudScheduler.Config.RabbitmqUsername,
		)
		csb.cloudScheduler.GoalManager.SetRMQHandler(
			interfacing.NewRabbitMQHandler(
				csb.cloudScheduler.Config.RabbitmqURI,
				csb.cloudScheduler.Config.RabbitmqUsername,
				csb.cloudScheduler.Config.RabbitmqPassword,
				"",
			),
		)
	}
	csb.cloudScheduler.GoalManager.Notifier.Subscribe(csb.cloudScheduler.chanFromGoalManager)
	return csb
}

func (csb *CloudSchedulerBuilder) AddAPIServer() *CloudSchedulerBuilder {
	csb.cloudScheduler.APIServer = &APIServer{
		cloudScheduler:         csb.cloudScheduler,
		version:                csb.cloudScheduler.Version,
		port:                   csb.cloudScheduler.Config.Port,
		enablePushNotification: csb.cloudScheduler.Config.PushNotification,
		subscribers:            make(map[string]map[chan *datatype.Event]bool),
	}
	return csb
}

func (rns *CloudSchedulerBuilder) Build() *CloudScheduler {
	return rns.cloudScheduler
}
