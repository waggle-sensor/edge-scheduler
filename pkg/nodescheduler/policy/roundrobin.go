package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type RoundRobinSchedulingPolicy struct {
}

func NewRoundRobinSchedulingPolicy() *RoundRobinSchedulingPolicy {
	return &RoundRobinSchedulingPolicy{}
}

// SelectBestPlugins returns the best plugin to run at the time
// It returns the oldest plugin amongst "ready" plugins
func (rs *RoundRobinSchedulingPolicy) SelectBestPlugins(scienceGoals map[string]*datatype.ScienceGoal, availableResource datatype.Resource, nodeID string) (pluginsToRun []*datatype.Plugin, err error) {
	var selectedPlugin *datatype.Plugin
	for _, goal := range scienceGoals {
		subGoal := goal.GetMySubGoal(nodeID)
		for _, plugin := range subGoal.Plugins {
			// If any plugin is currently running, we don't return other plugins to schedule
			if plugin.Status.SchedulingStatus == datatype.Running {
				return pluginsToRun, nil
			}
			// Pick up the oldest Ready plugin
			if plugin.Status.SchedulingStatus == datatype.Ready {
				// pluginsToRun = append(pluginsToRun, plugin)
				if selectedPlugin == nil {
					selectedPlugin = plugin
				} else if selectedPlugin.Status.Since.After(plugin.Status.Since) {
					selectedPlugin = plugin
				}
			}
		}
	}
	return []*datatype.Plugin{selectedPlugin}, nil
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
