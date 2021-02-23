package nodescheduler

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/knowledgebase"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler/policy"
)

const (
	maxChannelBuffer = 100
)

type NodeScheduler struct {
	ResourceManager             *ResourceManager
	Knowledgebase               *knowledgebase.Knowledgebase
	GoalManager                 *GoalManager
	chanContextEventToScheduler chan datatype.EventPluginContext
	chanFromGoalManager         chan *datatype.ScienceGoal
	chanRunGoal                 chan *datatype.ScienceGoal
	chanStopPlugin              chan *datatype.Plugin
	chanPluginToK3SClient       chan *datatype.Plugin
}

func NewNodeScheduler(rm *ResourceManager, kb *knowledgebase.Knowledgebase, gm *GoalManager) (*NodeScheduler, error) {
	return &NodeScheduler{
		ResourceManager:             rm,
		Knowledgebase:               kb,
		GoalManager:                 gm,
		chanContextEventToScheduler: make(chan datatype.EventPluginContext, maxChannelBuffer),
		chanFromGoalManager:         make(chan *datatype.ScienceGoal, maxChannelBuffer),
		chanRunGoal:                 make(chan *datatype.ScienceGoal, maxChannelBuffer),
		chanStopPlugin:              make(chan *datatype.Plugin, maxChannelBuffer),
		chanPluginToK3SClient:       make(chan *datatype.Plugin, maxChannelBuffer),
	}, nil
}

// Run handles communications between components for scheduling
func (ns *NodeScheduler) Run() {
	go ns.GoalManager.Run(ns.chanFromGoalManager)
	go ns.Knowledgebase.Run(ns.chanContextEventToScheduler)
	go ns.ResourceManager.Run(ns.chanPluginToK3SClient)

	for {
		select {
		case contextEvent := <-ns.chanContextEventToScheduler:
			scienceGoal, err := ns.GoalManager.GetScienceGoal(contextEvent.GoalID)
			if err != nil {
				logger.Error.Printf("%s", err.Error())
				continue
			}
			subGoal := scienceGoal.GetMySubGoal(ns.GoalManager.NodeID)
			err = subGoal.UpdatePluginContext(contextEvent)
			if err != nil {
				logger.Error.Printf("%s", err.Error())
				continue
			}
			// When a plugin becomes runnable see if it can be scheduled
			if contextEvent.Status == datatype.Runnable {
				ns.chanRunGoal <- scienceGoal
			} else if contextEvent.Status == datatype.Stoppable {
				ns.chanStopPlugin <- subGoal.GetPlugin(contextEvent.PluginName)
			}
		case scienceGoal := <-ns.chanFromGoalManager:
			ns.Knowledgebase.RegisterRules(scienceGoal, ns.GoalManager.NodeID)
		case scheduledScienceGoal := <-ns.chanRunGoal:
			logger.Info.Printf("Goal %s needs scheduling", scheduledScienceGoal.Name)
			subGoal := scheduledScienceGoal.GetMySubGoal(ns.GoalManager.NodeID)
			pluginsSubjectToSchedule := subGoal.GetSchedulablePlugins()
			logger.Info.Printf("Plugins subject to run: %v", pluginsSubjectToSchedule)
			// TODO: Resource model is not applied here -- needs improvements
			orderedPluginsToRun := policy.SimpleSchedulingPolicy(pluginsSubjectToSchedule, datatype.Resource{
				CPU:       999999,
				Memory:    999999,
				GPUMemory: 999999,
			})
			logger.Info.Printf("Ordered plugins subject to run: %v", orderedPluginsToRun)
			// Launch plugins
			for _, plugin := range orderedPluginsToRun {
				plugin.Status.SchedulingStatus = datatype.Running
				ns.chanPluginToK3SClient <- plugin
				logger.Info.Printf("Plugin %s has been scheduled to run", plugin.Name)
			}
			// // Launch plugins
			// if launchPlugins(schedulablePluginConfigs, pluginsToRun) {
			// 	// Track the plugin
			// 	// TODO: Later get status from k3s to track running plugins
			// 	currentPlugins = append(currentPlugins, pluginsToRun...)
			// }
			// logger.Info.Print("======================================")
			// scheduleTriggered = false
		case pluginToStop := <-ns.chanStopPlugin:
			if pluginToStop.Status.SchedulingStatus == datatype.Running {
				pluginToStop.Status.SchedulingStatus = datatype.Stopped
				ns.chanPluginToK3SClient <- pluginToStop
				logger.Info.Printf("Plugin %s has been triggered to stop", pluginToStop.Name)
			}
		}
	}
}
