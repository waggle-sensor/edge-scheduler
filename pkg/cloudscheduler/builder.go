package cloudscheduler

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

type CloudSchedulerConfig struct {
	Name                          string `json:"name" yaml:"name"`
	Version                       string
	NoRabbitMQ                    bool   `json:"no_rabbitmq" yaml:"noRabbitMQ"`
	RabbitmqURL                   string `json:"rabbitmq_url" yaml:"rabbiMQURL"`
	RabbitmqUsername              string `json:"rabbitmq_username" yaml:"rabbitMQUsername"`
	RabbitmqPassword              string `json:"rabbitmq_password" yaml:"rabbitMQPassword"`
	RabbitmqCaCertPath            string `json:"rabbitmq_cacert_path" yaml:"rabbitMQCacertPath"`
	ECRURI                        string `json:"ecr_uri" yaml:"ecrURI"`
	Port                          int    `json:"port" yaml:"port"`
	ManagementPort                int    `json:"management_port" yaml:"managementPort"`
	DataDir                       string `json:"data_dir,omitempty" yaml:"dataDir,omitempty"`
	PushNotification              bool   `json:"push_notification" yaml:"PushNotification"`
	AuthServerURL                 string `json:"auth_server_url" yaml:"authServerURL"`
	AuthToken                     string `json:"auth_token" yaml:"authToken"`
	JobReevaluationIntervalSecond int    `json:"job_reevaluation_interval_second" yaml:"jobReevaluationIntervalSecond"`
	Debug                         bool   `json:"debug" yaml:"debug"`
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
	csb.cloudScheduler.GoalManager.Notifier.Subscribe(csb.cloudScheduler.chanFromGoalManager)
	return csb
}

func (csb *CloudSchedulerBuilder) AddAPIServer() *CloudSchedulerBuilder {
	csb.cloudScheduler.APIServer = &APIServer{
		cloudScheduler:         csb.cloudScheduler,
		version:                csb.cloudScheduler.Version,
		port:                   csb.cloudScheduler.Config.Port,
		managementPort:         csb.cloudScheduler.Config.ManagementPort,
		enablePushNotification: csb.cloudScheduler.Config.PushNotification,
		subscribers:            make(map[string]map[chan *datatype.Event]bool),
		authenticator:          NewAuthenticator(csb.cloudScheduler.Config.AuthServerURL, csb.cloudScheduler.Config.AuthToken),
	}
	return csb
}

func (rns *CloudSchedulerBuilder) Build() *CloudScheduler {
	return rns.cloudScheduler
}
