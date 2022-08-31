package nodescheduler

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler/policy"
)

const (
	maxChannelBuffer = 100
)

type NodeScheduler struct {
	Version                     string
	NodeID                      string
	Config                      *NodeSchedulerConfig
	ResourceManager             *ResourceManager
	Knowledgebase               *KnowledgeBase
	GoalManager                 *NodeGoalManager
	APIServer                   *APIServer
	SchedulingPolicy            policy.SchedulingPolicy
	LogToBeehive                *interfacing.RabbitMQHandler
	chanContextEventToScheduler chan datatype.EventPluginContext
	chanFromGoalManager         chan datatype.Event
	chanFromResourceManager     chan datatype.Event
	chanFromCloudScheduler      chan *datatype.Event
	chanRunGoal                 chan *datatype.ScienceGoal
	chanStopPlugin              chan *datatype.Plugin
	chanPluginToResourceManager chan *datatype.Plugin
	chanNeedScheduling          chan datatype.Event
	chanAPIServerToGoalManager  chan *datatype.ScienceGoal
}

// func NewNodeScheduler(simulate bool) &NodeScheduler {
// 	// schedulingPolicy := policy.NewSimpleSchedulingPolicy()
// 	return &NodeScheduler{
// 		Simulate:                    simulate,
// 		SchedulingPolicy:            schedulingPolicy,
// 		chanContextEventToScheduler: make(chan datatype.EventPluginContext, maxChannelBuffer),
// 		chanFromGoalManager:         make(chan datatype.Event, maxChannelBuffer),
// 		chanRunGoal:                 make(chan *datatype.ScienceGoal, maxChannelBuffer),
// 		chanStopPlugin:              make(chan *datatype.Plugin, maxChannelBuffer),
// 		chanPluginToResourceManager: make(chan *datatype.Plugin, maxChannelBuffer),
// 		chanNeedScheduling:          make(chan string, 1),
// 		chanAPIServerToGoalManager:  make(chan *datatype.ScienceGoal, maxChannelBuffer),
// 	}
// }

// Configure sets up the followings in Kubernetes cluster
//
// - "ses" namespace
//
// - "wes-rabbitmq" and "wes-audio-server" services available in "ses" namespace
//
// - "waggle-data-config" and "wes-audio-server-plugin-conf" configmaps
//
// - "wes-ses-goal" configmap that accepts user goals
func (ns *NodeScheduler) Configure() (err error) {
	if ns.Config.Simulate {
		return
	}
	err = ns.ResourceManager.ConfigureKubernetes(ns.Config.InCluster, ns.Config.Kubeconfig)
	if err != nil {
		return
	}
	err = ns.ResourceManager.Configure()
	if err != nil {
		return
	}
	if ns.Config.GoalStreamURL != "" {
		logger.Info.Printf("Subscribing goal downstream from %s", ns.Config.GoalStreamURL)
		u, err := url.Parse(ns.Config.GoalStreamURL)
		if err != nil {
			return err
		}
		s := interfacing.NewHTTPRequest(u.Scheme + "://" + u.Host)
		s.Subscribe(u.Path, ns.chanFromCloudScheduler, true)
	}
	return
}

// Run handles communications between components for scheduling
func (ns *NodeScheduler) Run() {
	go ns.GoalManager.Run(ns.chanFromGoalManager)
	// go ns.Knowledgebase.Run()
	go ns.ResourceManager.Run(ns.chanPluginToResourceManager)
	go ns.APIServer.Run()

	// TODO: We generate a 30-second timer to (re)-evaluate given science rules
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			logger.Debug.Print("Rule evaluation triggered")
			triggerScheduling := false
			// ns.Knowledgebase.AddRawMeasure("sys.time.minute", time.Now().Minute())
			for goalID, _ := range ns.GoalManager.ScienceGoals {
				r, err := ns.Knowledgebase.EvaluateGoal(goalID)
				if err != nil {
					logger.Debug.Printf("Failed to evaluate goal %q: %s", goalID, err.Error())
				} else {
					for _, pluginName := range r {
						sg, err := ns.GoalManager.GetScienceGoalByID(goalID)
						if err != nil {
							logger.Debug.Printf("Failed to find goal %q: %s", goalID, err.Error())
						} else {
							plugin := sg.GetMySubGoal(ns.NodeID).GetPlugin(pluginName)
							if plugin.Status.SchedulingStatus == datatype.Waiting {
								plugin.UpdatePluginSchedulingStatus(datatype.Ready)
								triggerScheduling = true
							}
						}
					}
				}
			}
			if triggerScheduling {
				response := datatype.NewEventBuilder(datatype.EventPluginStatusPromoted).AddReason("kb triggered").Build()
				ns.chanNeedScheduling <- response
			}

		// case contextEvent := <-ns.chanContextEventToScheduler:
		// 	scienceGoal, err := ns.GoalManager.GetScienceGoal(contextEvent.GoalID)
		// 	if err != nil {
		// 		logger.Error.Printf("%s", err.Error())
		// 		continue
		// 	}
		// 	subGoal := scienceGoal.GetMySubGoal(ns.GoalManager.NodeID)
		// 	err = subGoal.UpdatePluginContext(contextEvent)
		// 	if err != nil {
		// 		logger.Error.Printf("%s", err.Error())
		// 		continue
		// 	}
		// 	// When a plugin becomes runnable see if it can be scheduled
		// 	if contextEvent.Status == datatype.Runnable {
		// 		ns.chanRunGoal <- scienceGoal
		// 	} else if contextEvent.Status == datatype.Stoppable {
		// 		ns.chanStopPlugin <- subGoal.GetPlugin(contextEvent.PluginName)
		// 	}
		case event := <-ns.chanFromGoalManager:
			// ns.Knowledgebase.RegisterRules(scienceGoal, ns.GoalManager.NodeID)
			logger.Debug.Printf("%s: %q", event.ToString(), event.GetGoalName())
			switch event.Type {
			case datatype.EventGoalStatusReceived, datatype.EventGoalStatusUpdated:
				sg, err := ns.GoalManager.GetScienceGoalByID(event.GetGoalID())
				if err != nil {
					logger.Error.Printf("Failed to find goal %q: %s", event.GetGoalID(), err.Error())
				} else {
					ns.Knowledgebase.AddRulesFromScienceGoal(sg)
					ns.chanNeedScheduling <- event
					go ns.LogToBeehive.SendWaggleMessage(event.ToWaggleMessage(), "all")
				}
			case datatype.EventGoalStatusRemoved:
				// TODO: Clean up plugins associated to the goal
				go ns.LogToBeehive.SendWaggleMessage(event.ToWaggleMessage(), "all")
			}
		case event := <-ns.chanFromResourceManager:
			logger.Debug.Printf("%s", event.ToString())
			switch event.Type {
			case datatype.EventPluginStatusLaunched:
				scienceGoal, err := ns.GoalManager.GetScienceGoalByID(event.GetGoalID())
				if err != nil {
					logger.Error.Printf("Could not get goal to update plugin status: %q", err.Error())
				} else {
					pluginName := event.GetPluginName()
					plugin := scienceGoal.GetMySubGoal(ns.NodeID).GetPlugin(pluginName)
					if plugin != nil {
						go ns.LogToBeehive.SendWaggleMessage(event.ToWaggleMessage(), "all")
					}
				}
			case datatype.EventPluginStatusComplete:
				// publish plugin completion message locally so that
				// rule checker knows when the last execution was
				// TODO: The message takes time to get into DB so the rule checker may not notice
				//       it if the checker is called before the delivery. We will need to make sure
				//       the message is delivered before triggering rule checking.
				pluginName := event.GetPluginName()
				message := datatype.NewMessage(
					string(datatype.EventPluginLastExecution),
					pluginName,
					event.Timestamp,
					map[string]string{},
				)
				go ns.LogToBeehive.SendWaggleMessage(message, "node")
				fallthrough
			case datatype.EventPluginStatusFailed:
				scienceGoal, err := ns.GoalManager.GetScienceGoalByID(event.GetGoalID())
				if err != nil {
					logger.Error.Printf("Could not get goal to update plugin status: %q", err.Error())
				} else {
					pluginName := event.GetPluginName()
					plugin := scienceGoal.GetMySubGoal(ns.NodeID).GetPlugin(pluginName)
					if plugin != nil {
						plugin.UpdatePluginSchedulingStatus(datatype.Waiting)
						go ns.LogToBeehive.SendWaggleMessage(event.ToWaggleMessage(), "all")
					}
				}
				ns.chanNeedScheduling <- event
			case datatype.EventFailure:
				logger.Debug.Printf("Error reported from resource manager: %q", event.GetReason())
				go ns.LogToBeehive.SendWaggleMessage(event.ToWaggleMessage(), "all")
				ns.chanNeedScheduling <- event
			case datatype.EventGoalStatusReceivedBulk:
				logger.Debug.Printf("A bulk goal is received")
				ns.ResourceManager.CleanUp()
				data := event.GetEntry("goals")
				var goals []datatype.ScienceGoal
				err := json.Unmarshal([]byte(data), &goals)
				if err != nil {
					logger.Error.Printf("Failed to load bulk goals %q", err.Error())
				} else {
					ns.GoalManager.SetGoals(goals)
				}
			}
			// case scheduledScienceGoal := <-ns.chanRunGoal:
			// 	logger.Info.Printf("Goal %s needs scheduling", scheduledScienceGoal.Name)
			// 	subGoal := scheduledScienceGoal.GetMySubGoal(ns.GoalManager.NodeID)
			// 	pluginsSubjectToSchedule := subGoal.GetSchedulablePlugins()
			// 	logger.Info.Printf("Plugins subject to run: %v", pluginsSubjectToSchedule)
			// 	// TODO: Resource model is not applied here -- needs improvements
			// 	orderedPluginsToRun := policy.SimpleSchedulingPolicy(pluginsSubjectToSchedule, datatype.Resource{
			// 		CPU:       999999,
			// 		Memory:    999999,
			// 		GPUMemory: 999999,
			// 	})
			// 	logger.Debug.Printf("Ordered plugins subject to run: %v", orderedPluginsToRun)
			// 	// Launch plugins
			// 	for _, plugin := range orderedPluginsToRun {
			// 		plugin.Status.SchedulingStatus = datatype.Running
			// 		ns.chanPluginToResourceManager <- plugin
			// 		logger.Info.Printf("Plugin %s has been scheduled to run", plugin.Name)
			// 	}
			// 	// // Launch plugins
			// 	// if launchPlugins(schedulablePluginConfigs, pluginsToRun) {
			// 	// 	// Track the plugin
			// 	// 	// TODO: Later get status from k3s to track running plugins
			// 	// 	currentPlugins = append(currentPlugins, pluginsToRun...)
			// 	// }
			// 	// logger.Info.Print("======================================")
			// 	// scheduleTriggered = false
			// case pluginToStop := <-ns.chanStopPlugin:
			// 	if pluginToStop.Status.SchedulingStatus == datatype.Running {
			// 		pluginToStop.Status.SchedulingStatus = datatype.Stopped
			// 		ns.chanPluginToK3SClient <- pluginToStop
			// 		logger.Info.Printf("Plugin %s has been triggered to stop", pluginToStop.Name)
			// 	}
		case event := <-ns.chanFromCloudScheduler:
			logger.Debug.Printf("%s", event.ToString())
			goals := event.GetEntry("goals")
			err := ns.ResourceManager.CreateConfigMap(
				configMapNameForGoals,
				map[string]string{"goals": goals},
				"default",
				true,
			)
			if err != nil {
				logger.Error.Printf("Failed to update goals for event %q", event.Type)
			}
		case event := <-ns.chanNeedScheduling:
			logger.Debug.Printf("Reason for (re)scheduling %q", event.Type)
			// Main logic: round robin + FIFO
			// Promote any waiting plugins
			// for id, goal := range ns.GoalManager.ScienceGoals {
			// 	logger.Debug.Printf("Checking Goal %q (%s)", goal.Name, id)
			// 	subGoal := goal.GetMySubGoal(ns.GoalManager.NodeID)
			// 	events := ns.SchedulingPolicy.SimpleScheduler.PromotePlugins(subGoal)
			// 	for _, e := range events {
			// 		logger.Debug.Printf("%s: %q", e.ToString(), e.GetPluginName())
			// 		go ns.LogToBeehive.SendWaggleMessage(e.ToWaggleMessage(), "all")
			// 	}
			// }
			// Select the best task
			plugins, err := ns.SchedulingPolicy.SelectBestPlugins(
				ns.GoalManager.ScienceGoals,
				datatype.Resource{
					CPU:       "999000m",
					Memory:    "999999Gi",
					GPUMemory: "999999Gi",
				},
				ns.GoalManager.NodeID,
			)
			if err != nil {
				logger.Error.Printf("Failed to get the best task to run %q", err.Error())
			} else {
				for _, plugin := range plugins {
					// if ns.ResourceManager.WillItFit(plugin) {
					e := datatype.NewEventBuilder(datatype.EventPluginStatusScheduled).AddReason("Fits to resource").AddPluginMeta(plugin).Build()
					logger.Debug.Printf("%s: %q (%q)", e.ToString(), e.GetPluginName(), e.GetReason())
					go ns.LogToBeehive.SendWaggleMessage(e.ToWaggleMessage(), "all")
					plugin.UpdatePluginSchedulingStatus(datatype.Running)
					go ns.ResourceManager.LaunchAndWatchPlugin(plugin)
					// } else {
					// 	logger.Debug.Printf("Resource is not availble for plugin %q", plugin.Name)
					// }
				}
			}
		}
	}
}
