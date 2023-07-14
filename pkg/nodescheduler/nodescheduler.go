package nodescheduler

import (
	"encoding/json"
	"net/url"
	"sync"
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
	mu                          sync.Mutex
	Version                     string
	NodeID                      string
	Config                      *NodeSchedulerConfig
	ResourceManager             *ResourceManager
	Knowledgebase               *KnowledgeBase
	GoalManager                 *NodeGoalManager
	APIServer                   *APIServer
	SchedulingPolicy            policy.SchedulingPolicy
	LogToBeehive                *interfacing.RabbitMQHandler
	ToScoreboard                *interfacing.RedisClient
	waitingQueue                datatype.Queue
	readyQueue                  datatype.Queue
	scheduledPlugins            datatype.Queue
	chanContextEventToScheduler chan datatype.EventPluginContext
	chanFromResourceManager     chan datatype.Event
	chanFromCloudScheduler      chan *datatype.Event
	chanNeedScheduling          chan datatype.Event
}

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
		logger.Info.Printf("subscribing goal downstream from %s", ns.Config.GoalStreamURL)
		u, err := url.Parse(ns.Config.GoalStreamURL)
		if err != nil {
			return err
		}
		s := interfacing.NewHTTPRequest(u.Scheme + "://" + u.Host)
		s.Subscribe(u.Path, ns.chanFromCloudScheduler, true)
	}
	if ns.LogToBeehive != nil {
		logger.Info.Println("starting THE RMQ handler loop for message publishing")
		ns.LogToBeehive.StartLoop()
	}
	return
}

// Run handles communications between components for scheduling
func (ns *NodeScheduler) Run() {
	logger.Info.Println("Attempting to clean up all plugins before starting scheduling...")
	ns.ResourceManager.CleanUp()
	go ns.ResourceManager.Run()
	go ns.APIServer.Run()
	ruleCheckingTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case event := <-ns.chanFromCloudScheduler:
			logger.Debug.Printf("%s", event.ToString())
			goals := event.GetEntry("goals").(string)
			err := ns.ResourceManager.CreateConfigMap(
				configMapNameForGoals,
				map[string]string{"goals": goals},
				"default",
				true,
			)
			if err != nil {
				logger.Error.Printf("Failed to update goals for event %q", event.Type)
			}
		case <-ruleCheckingTicker.C:
			logger.Debug.Print("Rule evaluation triggered")
			triggerScheduling := false
			// for goalID, _ := range ns.waitingQueue.GetGoalIDs() {
			// NOTE: Getting only goals of the plugins from the ready queue is useful only for scheduling action.
			//       To accommodate other types of action (i.e. publishing data to beehive) we need to
			//       evaluate all science rules no matter what plugins in the waiting queue.
			for goalID, sg := range ns.GoalManager.ScienceGoals {
				validRules, err := ns.Knowledgebase.EvaluateGoal(goalID)
				if err != nil {
					logger.Error.Printf("Failed to evaluate goal %q: %s", goalID, err.Error())
				} else {
					for _, r := range validRules {
						logger.Debug.Printf("Science rule %q is valid", r)
						switch r.ActionType {
						case datatype.ScienceRuleActionSchedule:
							// TODO: We will need to find a way to pass parameters to the plugin
							//       For example, schedule(plugin-a, duration=5m) <<
							pluginName := r.ActionObject
							plugin := sg.GetMySubGoal(ns.NodeID).GetPlugin(pluginName)
							if plugin == nil {
								logger.Error.Printf("failed to promote plugin: plugin name %q does not exist in goal %q", pluginName, goalID)
							} else {
								// make a hard copy of the plugin
								_p := *plugin
								pr := datatype.NewPluginRuntimeWithScienceRule(_p, *r)
								// TODO: we enable plugin-controller always. we will want to control this later.
								pr.SetPluginController(true)
								if _pr := ns.waitingQueue.Pop(pr); _pr != nil {
									ns.readyQueue.Push(pr)
									triggerScheduling = true
									logger.Debug.Printf("Plugin %s is promoted by rules", plugin.Name)
								}
							}
						case datatype.ScienceRuleActionPublish:
							eventName := r.ActionObject
							var value interface{}
							if v, found := r.ActionParameters["value"]; found {
								value = v
							} else {
								value = 1.
							}
							message := datatype.NewMessage(eventName, value, time.Now().UnixNano(), nil)
							var to string
							if v, found := r.ActionParameters["to"]; found {
								to = v
							} else {
								to = "all"
							}
							ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message, to)
						case datatype.ScienceRuleActionSet:
							stateName := r.ActionObject
							var value interface{}
							if v, found := r.ActionParameters["value"]; found {
								value = v
							} else {
								value = 1.
							}
							go func() {
								err := ns.ToScoreboard.Set(stateName, value)
								if err != nil {
									logger.Error.Printf("Failed to set %q: %s", stateName, err.Error())
								}
							}()
						}
					}
				}
			}
			if triggerScheduling {
				response := datatype.NewEventBuilder(datatype.EventPluginStatusPromoted).AddReason("kb triggered").Build()
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(response.ToWaggleMessage(), "node")
				ns.chanNeedScheduling <- response
			}
		case event := <-ns.chanNeedScheduling:
			logger.Debug.Printf("Reason for (re)scheduling %q", event.Type)
			logger.Debug.Printf("Plugins in ready queue: %+v", ns.readyQueue.GetPluginNames())
			// Select the best task
			pluginsToRun, err := ns.SchedulingPolicy.SelectBestPlugins(
				&ns.readyQueue,
				&ns.scheduledPlugins,
				datatype.Resource{
					CPU:       "999000m",
					Memory:    "999999Gi",
					GPUMemory: "999999Gi",
				},
			)
			if err != nil {
				logger.Error.Printf("Failed to get the best task to run %q", err.Error())
			} else {
				for _, _pr := range pluginsToRun {
					e := datatype.NewEventBuilder(datatype.EventPluginStatusScheduled).AddReason("Fit to resource").AddPluginMeta(&_pr.Plugin).Build()
					logger.Debug.Printf("%s: %q (%q)", e.ToString(), e.GetPluginName(), e.GetReason())
					ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
					pr := ns.readyQueue.Pop(_pr)
					ns.scheduledPlugins.Push(pr)
					go ns.ResourceManager.LaunchAndWatchPlugin(pr)
				}
			}
		case event := <-ns.chanFromResourceManager:
			logger.Debug.Printf("%s", event.ToString())
			switch event.Type {
			case datatype.EventPluginStatusLaunched:
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(event.ToWaggleMessage(), "all")
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
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message, "node")
				fallthrough
			case datatype.EventPluginStatusFailed:
				scienceGoal, err := ns.GoalManager.GetScienceGoalByID(event.GetGoalID())
				if err != nil {
					logger.Error.Printf("Could not get goal to update plugin status: %q", err.Error())
				} else {
					pluginName := event.GetPluginName()
					if plugin := scienceGoal.GetMySubGoal(ns.NodeID).GetPlugin(pluginName); plugin != nil {
						p := *plugin
						_pr := &datatype.PluginRuntime{
							Plugin: p,
						}
						pr := ns.scheduledPlugins.Pop(_pr)
						ns.waitingQueue.Push(pr)
						ns.LogToBeehive.SendWaggleMessageOnNodeAsync(event.ToWaggleMessage(), "all")
					}
				}
				// We trigger the scheduling logic for plugins that need to run
				ns.chanNeedScheduling <- event
			case datatype.EventGoalStatusReceivedBulk:
				// A goal set is received. We add or update the goals.
				logger.Debug.Printf("A bulk goal is received")
				data := event.GetEntry("goals").(string)
				var goals []datatype.ScienceGoal
				err := json.Unmarshal([]byte(data), &goals)
				if err != nil {
					logger.Error.Printf("Failed to load bulk goals %q", err.Error())
				} else {
					ns.handleBulkGoals(goals)
				}
			}
		}
	}
}

func (ns *NodeScheduler) registerGoal(goal *datatype.ScienceGoal) {
	ns.GoalManager.AddGoal(goal)
	if mySubGoal := goal.GetMySubGoal(ns.NodeID); mySubGoal == nil {
		logger.Error.Printf("Failed to find my sub goal from science goal %q. Failed to register the goal.", goal.ID)
	} else {
		err := ns.Knowledgebase.AddRulesFromScienceGoal(goal)
		if err != nil {
			logger.Error.Printf("Failed to add science rules of goal %q: %s", goal.ID, err.Error())
		}
		for _, p := range mySubGoal.GetPlugins() {
			// copy plugin object
			_p := *p
			ns.waitingQueue.Push(&datatype.PluginRuntime{
				Plugin: _p,
			})
			logger.Debug.Printf("plugin %s is added to the watiting queue", p.Name)
		}
	}
}

func (ns *NodeScheduler) cleanUpGoal(goal *datatype.ScienceGoal) {
	ns.Knowledgebase.DropRules(goal.Name)
	for _, p := range goal.GetMySubGoal(ns.NodeID).GetPlugins() {
		_p := *p
		pr := &datatype.PluginRuntime{
			Plugin: _p,
		}
		if a := ns.waitingQueue.Pop(pr); a != nil {
			logger.Debug.Printf("plugin %s is removed from the waiting queue", p.Name)
		}
		if a := ns.readyQueue.Pop(pr); a != nil {
			logger.Debug.Printf("plugin %s is removed from the ready queue", p.Name)
		}
		if a := ns.scheduledPlugins.Pop(pr); a != nil {
			if pod, err := ns.ResourceManager.GetPod(a.Plugin.Name); err != nil {
				logger.Error.Printf("Failed to get pod of the plugin %q", a.Plugin.Name)
			} else {
				e := datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddPluginMeta(&a.Plugin).AddPodMeta(pod).AddReason("Cleaning up the plugin due to deletion of the goal").Build()
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
			}
			ns.ResourceManager.RemovePlugin(&a.Plugin)
			logger.Debug.Printf("plugin %s is removed from running", p.Name)

		}
	}
	ns.GoalManager.DropGoal(goal.ID)
}

// handleBulkGoals adds or updates each goal in given goal list
func (ns *NodeScheduler) handleBulkGoals(goals []datatype.ScienceGoal) {
	// NOTE: There are multiple triggers that call this function
	//       For example, k3s configmap change for goals and cloud scheduler pushing
	//       new goals. We mutex lock this to secure adding/dropping goals.
	ns.mu.Lock()
	defer ns.mu.Unlock()
	goalsToKeep := make(map[string]bool)
	for _, goal := range goals {
		if subGoal := goal.GetMySubGoal(ns.NodeID); subGoal != nil {
			subGoal.AddChecksum()
		}
		goalsToKeep[goal.ID] = true
		if existingGoal, _ := ns.GoalManager.GetScienceGoalByJobID(goal.JobID); existingGoal != nil {
			// We assume that if the goal ID are the same, the goal has not changed.
			if existingGoal.ID == goal.ID {
				logger.Info.Printf("The goal %s exists and no changes in the goal. Skipping adding the goal", goal.Name)
				continue
			} else {
				logger.Info.Printf("The goal %s %q exists and has changed its content. Cleaning up the existing goal %q", goal.Name, goal.ID, existingGoal.ID)
				ns.cleanUpGoal(existingGoal)
				ns.registerGoal(&goal)
				e := datatype.NewEventBuilder(datatype.EventGoalStatusUpdated).AddGoal(&goal).Build()
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
			}
		} else {
			logger.Info.Printf("Adding the new goal %s %q", goal.Name, goal.ID)
			ns.registerGoal(&goal)
			e := datatype.NewEventBuilder(datatype.EventGoalStatusReceived).AddGoal(&goal).Build()
			ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
		}
	}
	// Remove any existing goal that is not included in the new goal set
	for _, goal := range ns.GoalManager.ScienceGoals {
		if _, exist := goalsToKeep[goal.ID]; !exist {
			ns.cleanUpGoal(&goal)
			event := datatype.NewEventBuilder(datatype.EventGoalStatusRemoved).AddGoal(&goal).Build()
			ns.LogToBeehive.SendWaggleMessageOnNodeAsync(event.ToWaggleMessage(), "all")
		}
	}
}
