package nodescheduler

import (
	"time"

	"github.com/sagecontinuum/ses/pkg/logger"
)

var (
	chanTriggerSchedule = make(chan string)
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

func RunScheduler() {
	for {
		// Do scheduling only when needed
		select {
		case why := <-chanTriggerSchedule:
			logger.Info.Printf("A scheduling triggered: %s", why)
			// Ask KB what can run now
			schedulablePluginNames := Ask()
			logger.Info.Printf("What can run: %v", schedulablePluginNames)
			// Get the configs of the schedulable plugins
			schedulablePluginConfigs := getSchedulablePluginConfigs(schedulablePluginNames)
			// Decide what to run
			pluginsToRun := policy.NoSchedulingStrategy(schedulablePluginConfigs, currentPlugins)
			logger.Info.Printf("Things to deploy: %v", pluginsToRun)
			logger.Info.Printf("What has been running: %v", currentPlugins)
			// Terminate plugins that are not subject to run
			terminatePlugins(pluginsToRun)
			// Launch plugins
			if launchPlugins(schedulablePluginConfigs, pluginsToRun) {
				// Track the plugin
				// TODO: Later get status from k3s to track running plugins
				currentPlugins = append(currentPlugins, pluginsToRun...)
			}
			logger.Info.Print("======================================")
			// scheduleTriggered = false
		}
		time.Sleep(3 * time.Second)
	}
}

// RunNodeScheduler initializes itself and runs the main routine
func RunNodeScheduler() {
	InitializeKB()
	// InitializeK3s()
	InitializeGoalManager()

	// if !*dryRun {
	// 	InitializeMeasureCollector("localhost:5672")
	// 	go RunMeasureCollector(chanFromMeasure)
	// }

	go RunScheduler()
	// go RunKnowledgebase(chanFromMeasure, chanTriggerScheduler)
	go RunGoalManager()
	createRouter()
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
