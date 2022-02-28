package cloudscheduler

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
)

const maxChannelBuffer = 100

// CloudScheduler structs the cloud scheduler
type CloudScheduler struct {
	Name                string
	Version             string
	GoalManager         *CloudGoalManager
	Meta                *MetaHandler
	Validator           *JobValidator
	APIServer           *APIServer
	chanFromGoalManager chan datatype.Event
}

func (cs *CloudScheduler) Run() {
	go cs.APIServer.Run()
	logger.Info.Printf("Cloud Scheduler %s starts...", cs.Name)
	for {
		select {
		case event := <-cs.chanFromGoalManager:
			logger.Debug.Printf("%s: %q", event.ToString(), event.GetGoalName())
		}
	}
}
