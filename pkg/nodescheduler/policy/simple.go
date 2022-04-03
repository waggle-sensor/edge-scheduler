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

// SelectBestPlugins returns the best plugin to run at the time
// For SimpleSchedulingPolicy, it returns the oldest plugin amongst "ready" plugins
func (ss *SimpleSchedulingPolicy) SelectBestPlugins(scienceGoals map[string]*datatype.ScienceGoal, availableResource datatype.Resource, nodeID string) (pluginsToRun []*datatype.Plugin, err error) {
	// var selectedPlugin *datatype.Plugin
	for _, goal := range scienceGoals {
		subGoal := goal.GetMySubGoal(nodeID)
		for _, plugin := range subGoal.Plugins {
			if plugin.Status.SchedulingStatus == datatype.Ready {
				pluginsToRun = append(pluginsToRun, plugin)
				// if selectedPlugin == nil {
				// 	selectedPlugin = plugin
				// } else if selectedPlugin.Status.Since.After(plugin.Status.Since) {
				// 	selectedPlugin = plugin
				// }
			}
		}
	}
	return pluginsToRun, nil
}

// func (ss *SimpleSchedulingPolicy) PromotePlugins(subGoal *datatype.SubGoal) (events []datatype.Event) {
// 	for _, plugin := range subGoal.Plugins {
// 		if plugin.Status.SchedulingStatus == datatype.Waiting {
// 			plugin.UpdatePluginSchedulingStatus(datatype.Ready)
// 			events = append(events, datatype.NewEventBuilder(datatype.EventPluginStatusPromoted).AddPluginMeta(plugin).Build())
// 		}
// 	}
// 	return
// }
