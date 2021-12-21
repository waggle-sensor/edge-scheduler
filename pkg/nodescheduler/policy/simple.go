package policy

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
)

type SchedulingPolicy struct {
	SimpleScheduler *SimpleSchedulingPolicy
}

type SimpleSchedulingPolicy struct {
}

func NewSimpleSchedulingPolicy() *SchedulingPolicy {
	return &SchedulingPolicy{SimpleScheduler: &SimpleSchedulingPolicy{}}
}

// SelectBestTask returns the best plugin to run at the time
// For SimpleSchedulingPolicy, it returns the oldest plugin amongst "ready" plugins
func (ss *SimpleSchedulingPolicy) SelectBestTask(scienceGoals map[string]*datatype.ScienceGoal, availableResource datatype.Resource, nodeID string) (*datatype.Plugin, error) {
	var selectedPlugin *datatype.Plugin
	for _, goal := range scienceGoals {
		subGoal := goal.GetMySubGoal(nodeID)
		for _, plugin := range subGoal.Plugins {
			if plugin.Status.SchedulingStatus == datatype.Ready {
				if selectedPlugin == nil {
					selectedPlugin = plugin
				} else if selectedPlugin.Status.Since.After(plugin.Status.Since) {
					selectedPlugin = plugin
				}
			}
		}
	}
	return selectedPlugin, nil
}
