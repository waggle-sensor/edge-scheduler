package nodescheduler

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/knowledgebase"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler/policy"
)

var (
	chanContextEventToScheduler = make(chan datatype.EventPluginContext)
	chanJobQueue                = make(chan *datatype.ScienceGoal)
	chanPluginToK3SClient       = make(chan *datatype.Plugin)
)

// type K3sTemplate struct {
// 	Metadata struct {
// 		Labels struct {
// 			App string `yaml:"app"`
// 		}
// 	}
// 	Spec struct {
// 		Containers []Plugin `yaml:"containers,flow,omitempty"`
// 	}
// }
//
// type K3sPod struct {
// 	ApiVersion string `yaml:"apiVersion"`
// 	Kind string `yaml:"kind"`
// 	Metadata struct {
// 		Name string `yaml:"name"`
// 		Namespace string `yaml:"namespace"`
// 	}
// 	Spec struct {
// 		Selector struct {
// 			MatchLabels struct {
// 				App string `yaml:"app"`
// 			}
// 		}
// 		Template K3sTemplate `yaml:"template"`
// 	}
// }

// func RegisterGoal(goal Goal) {
// 	goals = append(goals, goal)
// 	// Add rules to KB
// 	for _, rule := range goal.Body.Rules {
// 		AddClause(rule)
// 	}
// }

// RunScheduler handles communications between components
func RunScheduler() {
	for {
		select {
		case contextEvent := <-chanContextEventToScheduler:
			scienceGoal, err := GetScienceGoal(contextEvent.GoalID)
			if err != nil {
				logger.Error.Printf("%s", err.Error())
				continue
			}
			subGoal := scienceGoal.GetMySubGoal(nodeID)
			err = subGoal.UpdatePluginContext(contextEvent)
			if err != nil {
				logger.Error.Printf("%s", err.Error())
				continue
			}
			// When a plugin becomes runnable see if it can be scheduled
			if contextEvent.Status == datatype.Runnable {
				chanJobQueue <- scienceGoal
			}
		case scheduledGoal := <-chanJobQueue:
			logger.Info.Printf("%s needs scheduling", scheduledGoal.Name)
			subGoal := scheduledGoal.GetMySubGoal(nodeID)
			pluginsSubjectToSchedule := subGoal.GetSchedulablePlugins()
			logger.Info.Printf("Plugins subject to run: %v", pluginsSubjectToSchedule)
			// TODO: Resource model is not applied here -- needs improvements
			orderedPluginsToRun := policy.SimpleSchedulingPolicy(pluginsSubjectToSchedule, datatype.Resource{
				CPU:       999999,
				Memory:    999999,
				GPUMemory: 999999,
			})
			logger.Info.Printf("Ordered plugins subject to run: %v", orderedPluginsToRun)

			// // Launch plugins
			// if launchPlugins(schedulablePluginConfigs, pluginsToRun) {
			// 	// Track the plugin
			// 	// TODO: Later get status from k3s to track running plugins
			// 	currentPlugins = append(currentPlugins, pluginsToRun...)
			// }
			// logger.Info.Print("======================================")
			// scheduleTriggered = false
		}
	}
}

// RunNodeScheduler initializes itself and runs the main routine
func RunNodeScheduler() {
	knowledgebase.InitializeKB(chanContextEventToScheduler)
	InitializeK3s(chanPluginToK3SClient)
	InitializeGoalManager()

	// if !*dryRun {
	// 	InitializeMeasureCollector("localhost:5672")
	// 	go RunMeasureCollector(chanFromMeasure)
	// }

	RunScheduler()
	// go RunKnowledgebase(chanFromMeasure, chanTriggerScheduler)
	// createRouter()
}

func findIndex(array []string, target string) int {
	for i, s := range array {
		if s == target {
			return i
		}
	}
	return -1
}

func removeItem(array []string, target string) []string {
	var index int
	index = -1
	for i, s := range array {
		if s == target {
			index = i
			break
		}
	}
	if index >= 0 {
		return append(array[:index], array[index+1:]...)
	} else {
		return array
	}
}

// func getSchedulablePluginConfigs(pluginNames []string) (pluginConfigs map[string]PluginConfig) {
// 	pluginConfigs = make(map[string]PluginConfig)
// 	for _, goal := range goals {
// 		for _, appConfig := range goal.Body.AppConfig {
// 			i := findIndex(pluginNames, appConfig.Name)
// 			if i >= 0 {
// 				pluginConfigs[appConfig.Name] = appConfig
// 			}
// 		}
// 	}
// 	return
// }
//
// func terminatePlugins(pluginsToRun []string) {
// 	for _, plugin := range currentPlugins {
// 		if findIndex(pluginsToRun, plugin) < 0 {
// 			if TerminatePlugin(plugin) {
// 				currentPlugins = removeItem(currentPlugins, plugin)
// 			}
// 		}
// 	}
// }
//
// func launchPlugins(pluginConfigs map[string]PluginConfig, pluginNames []string) bool {
// 	for _, pluginName := range pluginNames {
// 		// Skip launching if already running
// 		logger.Info.Printf("Finding index %v, %s", currentPlugins, pluginName)
// 		i := findIndex(currentPlugins, pluginName)
// 		logger.Info.Printf("Index %d", i)
// 		if i >= 0 {
// 			logger.Info.Printf("Already exists: %s", pluginName)
// 			continue
// 		} else {
// 			pluginConfig := pluginConfigs[pluginName]
// 			deployment := CreateK3sPod(pluginConfig)
// 			if LaunchPlugin(deployment) {
// 				return true
// 			} else {
// 				return false
// 			}
// 		}
// 	}
// 	return false
// }