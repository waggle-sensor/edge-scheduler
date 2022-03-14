package cloudscheduler

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/interfacing"
)

type RealCloudScheduler struct {
	cloudScheduler *CloudScheduler
}

func NewRealCloudSchedulerBuilder(name string, version string) *RealCloudScheduler {
	return &RealCloudScheduler{
		cloudScheduler: &CloudScheduler{
			Name:                name,
			Version:             version,
			Validator:           NewJobValidator(),
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
	rcs.cloudScheduler.GoalManager.Notifier.Subscribe(rcs.cloudScheduler.chanFromGoalManager)
	return rcs
}

func (rcs *RealCloudScheduler) AddAPIServer(port int) *RealCloudScheduler {
	rcs.cloudScheduler.APIServer = &APIServer{
		cloudScheduler: rcs.cloudScheduler,
		version:        rcs.cloudScheduler.Version,
		port:           port,
	}
	return rcs
}

func (rns *RealCloudScheduler) Build() *CloudScheduler {
	return rns.cloudScheduler
}
