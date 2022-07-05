package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type SchedulingPolicy interface {
	SelectBestPlugins(map[string]*datatype.ScienceGoal, datatype.Resource, string) ([]*datatype.Plugin, error)
}

func GetSchedulingPolicyByName(policyName string) SchedulingPolicy {
	switch policyName {
	case "default":
		return NewSimpleSchedulingPolicy()
	case "roundrobin":
		return NewRoundRobinSchedulingPolicy()
	default:
		return NewSimpleSchedulingPolicy()
	}
}

type SimpleSchedulingPolicy struct {
}

func NewSimpleSchedulingPolicy() *SimpleSchedulingPolicy {
	return &SimpleSchedulingPolicy{}
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
