package cloudscheduler

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type CloudSchedulerConfig struct {
	Name               string
	Version            string
	NoRabbitMQ         bool
	RabbitmqURI        string
	RabbitmqUsername   string
	RabbitmqPassword   string
	ECRURL             string
	Port               int
	DataDir            string
	NoPushNotification bool
}

type RealCloudScheduler struct {
	cloudScheduler *CloudScheduler
	config         *CloudSchedulerConfig
}

func NewRealCloudSchedulerBuilder(config *CloudSchedulerConfig) *RealCloudScheduler {
	return &RealCloudScheduler{
		cloudScheduler: &CloudScheduler{
			Name:                config.Name,
			Version:             config.Version,
			Config:              config,
			Validator:           NewJobValidator(config.DataDir),
			chanFromGoalManager: make(chan datatype.Event, maxChannelBuffer),
		},
	}
}

func (rcs *RealCloudScheduler) AddGoalManager() *RealCloudScheduler {
	rcs.cloudScheduler.GoalManager = &CloudGoalManager{
		scienceGoals: make(map[string]*datatype.ScienceGoal),
		Notifier:     interfacing.NewNotifier(),
		jobs:         make(map[string]*datatype.Job),
	}
	if !rcs.cloudScheduler.Config.NoRabbitMQ {
		logger.Info.Printf(
			"Using RabbitMQ at %s with user %s",
			rcs.cloudScheduler.Config.RabbitmqURI,
			rcs.cloudScheduler.Config.RabbitmqUsername,
		)
		rcs.cloudScheduler.GoalManager.SetRMQHandler(
			interfacing.NewRabbitMQHandler(
				rcs.cloudScheduler.Config.RabbitmqURI,
				rcs.cloudScheduler.Config.RabbitmqUsername,
				rcs.cloudScheduler.Config.RabbitmqPassword,
				"",
			),
		)
	}
	rcs.cloudScheduler.GoalManager.Notifier.Subscribe(rcs.cloudScheduler.chanFromGoalManager)
	return rcs
}

func (rcs *RealCloudScheduler) AddAPIServer() *RealCloudScheduler {
	rcs.cloudScheduler.APIServer = &APIServer{
		cloudScheduler:         rcs.cloudScheduler,
		version:                rcs.cloudScheduler.Version,
		port:                   rcs.cloudScheduler.Config.Port,
		enablePushNotification: rcs.cloudScheduler.Config.NoPushNotification,
		subscribers:            make(map[string]map[chan *datatype.Event]bool),
	}
	return rcs
}

func (rns *RealCloudScheduler) Build() *CloudScheduler {
	return rns.cloudScheduler
}
