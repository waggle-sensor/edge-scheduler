package nodescheduler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler/policy"
	v1 "k8s.io/api/core/v1"
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
	readyQueue                  datatype.Queue // act a job queue for resource management
	scheduledPlugins            datatype.Queue
	chanContextEventToScheduler chan datatype.EventPluginContext
	chanFromResourceManager     chan datatype.Event
	chanFromCloudScheduler      chan datatype.Event
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
	go ns.ResourceManager.Run()
	go ns.APIServer.Run()
	ruleCheckingTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case event := <-ns.chanFromCloudScheduler:
			e := event.(datatype.SchedulerEvent)
			goals := e.GetEntry("goals").(string)
			logger.Debug.Printf("%s: %s", e.ToString(), goals)
			err := ns.ResourceManager.CreateConfigMap(
				configMapNameForGoals,
				map[string]string{"goals": goals},
				ns.ResourceManager.Namespace,
				true,
			)
			if err != nil {
				logger.Error.Printf("Failed to update goals for event %q", e.Type)
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
							if pr := ns.GoalManager.GetPluginRuntime(PluginIndex{
								name:   pluginName,
								jobID:  sg.JobID,
								goalID: sg.ID,
							}); pr == nil {
								logger.Error.Printf("failed to promote plugin: plugin name %q for goal %q not registered", pluginName, goalID)
								// TODO: we may want to verify what exist and why this happens
							} else if !pr.IsState(datatype.Inactive) {
								logger.Debug.Printf("plugin %q is already active. no need to activate it", pr.Plugin.Name)
							} else {
								pr.UpdateWithScienceRule(r)
								// TODO: We disable the plugin controller until we actually use it.
								//       This causes problems of Pods not finishing and hanging in StartError
								// pr.SetPluginController(true)
								pr.GeneratePodInstance()
								pr.Queued()
								msg := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusQueued).
									AddPluginRuntimeMeta(*pr).
									AddPluginMeta(pr.Plugin).
									AddReason(fmt.Sprintf("triggered by %s", r.Condition)).
									Build().(datatype.SchedulerEvent)
								ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg.ToWaggleMessage(), "all")
								ns.readyQueue.Push(pr)
								triggerScheduling = true
								logger.Info.Printf("Plugin %s is queued by %s", pr.Plugin.Name, r.Condition)
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
				privateMessage := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusQueued).
					AddReason("kb triggered").
					Build().(datatype.SchedulerEvent)
				ns.chanNeedScheduling <- privateMessage
			}
		case event := <-ns.chanNeedScheduling:
			e := event.(datatype.SchedulerEvent)
			logger.Debug.Printf("Reason for (re)scheduling %q", e.Type)
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
					pluginEvent := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusSelected).
						AddReason("Fit to resource").
						AddPluginRuntimeMeta(*_pr).
						AddPluginMeta(_pr.Plugin).
						Build().(datatype.SchedulerEvent)
					logger.Debug.Printf("%s: %q (%q)", pluginEvent.ToString(), pluginEvent.GetPluginName(), pluginEvent.GetReason())
					ns.LogToBeehive.SendWaggleMessageOnNodeAsync(pluginEvent.ToWaggleMessage(), "all")
					pr := ns.readyQueue.Pop(_pr)
					ns.scheduledPlugins.Push(pr)
					go func() {
						// TODO: when failed we need to put the pr back to inactive...???
						logger.Debug.Printf("Running plugin %q...", pr.Plugin.Name)
						pod, err := ns.ResourceManager.CreatePodTemplate(pr)
						if err != nil {
							logger.Error.Printf("Failed to create Kubernetes Pod for %q: %q", pr.Plugin.Name, err.Error())
							msg := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
								AddPluginRuntimeMeta(*pr).
								AddReason(err.Error()).
								AddPluginMeta(pr.Plugin).
								Build().(datatype.SchedulerEvent)
							ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg.ToWaggleMessage(), "all")
							return
						}
						// we override the plugin name to distinguish the same plugin name from different jobs
						if pr.Plugin.JobID != "" {
							pod.SetName(fmt.Sprintf("%s-%s", pod.GetName(), pr.Plugin.JobID))
						}
						err = ns.ResourceManager.CreatePod(pod)
						// defer rm.TerminatePod(pod.Name)
						if err != nil {
							logger.Error.Printf("Failed to run %q: %q", pod.Name, err.Error())
							msg := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
								AddPluginRuntimeMeta(*pr).
								AddReason(err.Error()).
								AddPluginMeta(pr.Plugin).
								Build().(datatype.SchedulerEvent)
							ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg.ToWaggleMessage(), "all")
							if err = ns.ResourceManager.TerminatePod(pod.Name); err != nil {
								logger.Error.Printf("Failed to delete %s: %s", pod.Name, err.Error())
							} else {
								logger.Info.Printf("%s is deleted as it failed to run", pod.Name)
							}
							return
						}
						logger.Info.Printf("Plugin %q is created", pod.Name)
						pr.Plugin.PluginSpec.Job = pod.Name
					}()
				}
			}
		case event := <-ns.chanFromResourceManager:
			e := event.(KubernetesEvent)
			logger.Debug.Printf("Event received from Resource Manager: %s %q", e.Type, e.Action)
			switch e.Type {
			case KubernetesEventTypePod:
				ns.handleKubernetesPodEvent(e)
			case KubernetesEventTypeEvent:
				ns.handleKubernetesEventEvent(e)
			case KubernetesEventTypeConfigMap:
				ns.handleKubernetesConfigMapEvent(e)
			default:
				// We shouldn't receive unknown type event. If so, we need to implement it here
				panic(fmt.Sprintf("Unknown event received from Resource Manager: %s", e.Type))
			}
		}
	}
}

func (ns *NodeScheduler) handleKubernetesPodEvent(e KubernetesEvent) {
	pod := e.Pod
	logger.Debug.Printf("pod status: %s", string(pod.Status.Phase))
	for _, i := range pod.Status.InitContainerStatuses {
		logger.Debug.Printf("%s: (%s) %s", pod.Name, i.Name, &i.State)
	}
	for _, c := range pod.Status.ContainerStatuses {
		logger.Debug.Printf("%s: (%s) %s", pod.Name, c.Name, &c.State)
	}

	pluginName, pluginNameExist := pod.Labels[PodLabelPluginTask]
	goalID, goalIDExist := pod.Labels[PodLabelGoalID]
	jobID, jobIDExist := pod.Labels[PodLabelJobID]
	if !pluginNameExist || !goalIDExist || !jobIDExist {
		logger.Error.Printf("Pod %q labels do not have information for Plugin Runtime from Pod: %v", pod.Name, pod.Labels)
		return
	}
	pluginIndex := PluginIndex{
		name:   pluginName,
		goalID: goalID,
		jobID:  jobID,
	}
	pr := ns.GoalManager.GetPluginRuntime(pluginIndex)
	if pr == nil {
		logger.Error.Printf("Failed to find Plugin Runtime using index %v", pluginIndex)
		return
	}
	logger.Debug.Printf("current Plugin %q state %s", pod.Name, pr.Status.State)

	// we may receive Pod events from Kubernetes on already existing ones
	// TODO: we need to not sending messages to cloud about those already exist
	//       by skipping the following steps
	if !ns.scheduledPlugins.IsExist(pr) {
		// probably unmanaged one, we should ignore it
		logger.Info.Printf("pod %q has no associated Plugin in the queue. Ignoring the event", pod.Name)
		return
	}

	switch e.Action {
	case KubernetesEventTypeAdd:
		logger.Info.Printf("Plugin %q is scheduled", pod.Name)
		pr.Scheduled()
		pr.SetPodUID(string(pod.UID))
		msg := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusScheduled).
			AddPluginRuntimeMeta(*pr).
			AddPodMeta(pod).
			AddPluginMeta(pr.Plugin).
			Build().(datatype.SchedulerEvent)
		ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg.ToWaggleMessage(), "all")

		// NOTE: To support backward compatibility, we also send the "launched" event
		msg2 := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusLaunched).
			AddPluginRuntimeMeta(*pr).
			AddPodMeta(pod).
			AddPluginMeta(pr.Plugin).
			Build().(datatype.SchedulerEvent)
		ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg2.ToWaggleMessage(), "all")
	case KubernetesEventTypeModified:
		switch pod.Status.Phase {
		case v1.PodPending:
			// we expect init container running and completion
			if !pr.IsState(datatype.Initializing) {
				logger.Info.Printf("plugin %q is being initialized", pod.Name)
				pr.Initializing()
				msg := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusInitializing).
					AddPluginRuntimeMeta(*pr).
					AddPodMeta(pod).
					AddPluginMeta(pr.Plugin).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg.ToWaggleMessage(), "all")
			}
		case v1.PodRunning:
			// we expect the application container starts to run.
			// we may not consider the start of our plugin-controller container
			if pluginContainerStatus, err := ns.ResourceManager.GetContainerStatusFromPod(pod, pluginName); err != nil {
				// Failed to retrieve the status
				e := fmt.Sprintf("Failed to get container status: %s", err.Error())
				logger.Error.Println(e)
				pr.Failed()
				messageBuilder, err := ns.ResourceManager.AnalyzeFailureOfPod(pod)
				if err != nil {
					logger.Error.Println(err.Error())
				}
				message := messageBuilder.AddPluginRuntimeMeta(*pr).
					AddPodMeta(pod).
					AddPluginMeta(pr.Plugin).
					AddReason(e).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message.ToWaggleMessage(), "all")
				defer ns.ResourceManager.TerminatePod(pod.Name)
			} else if t := pluginContainerStatus.State.Terminated; t != nil {
				// Whenever the plugin container terminates that the plugin container
				// does not notice, we should stop!!!
				// NOTE: https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/
				//       adds a mechanism for sidecar container to exit when the main container exits.
				//       This behavior will save our effort on checking the condition, in this if statement
				// NOTE: The mechanism to safely terminate the plugin in this state is complicated.
				//       We will deal with this issue later.

				// pluginControllerContainerStatus := ns.ResourceManager.GetContainerStatusFromPod(pod, PluginControllerContainerName)
				// if pluginControllerContainerStatus.State.Terminated != nil {
				// 	logger.Error.Printf("plugin %q exited with return code %d, but the Pod remains as the plugin-controller still runs", pod.Name, t.ExitCode)
				// 	if t.ExitCode == 0 {
				// 		pr.Completed()
				// 	} else {
				// 		pr.Failed()
				// 	}
				// 	// NOTE: We would not want to terminate the pod because it can mislead users to
				// 	//       think the plugin is failed because of theirs.
				// 	defer ns.ResourceManager.TerminatePod(pod.Name)
				// }
			} else if pluginContainerStatus.State.Running != nil {
				logger.Info.Printf("Plugin %q starts to run", pod.Name)
				pr.Running()
				msg := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusRunning).
					AddPluginRuntimeMeta(*pr).
					AddPodMeta(pod).
					AddPluginMeta(pr.Plugin).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(msg.ToWaggleMessage(), "all")
			}
		case v1.PodSucceeded:
			// This event occurs when all containers finished successfully
			// NOTE: we receive multiple of this event for a Pod. We don't want to
			//       trigger message multiple times.
			if !pr.IsState(datatype.Completed) {
				logger.Info.Printf("Plugin %q succeeded", pod.Name)
				pr.Completed()
				// 	// publish plugin completion message locally so that
				// 	// rule checker knows when the last execution was
				// 	// TODO: The message takes time to get into DB so the rule checker may not notice
				// 	//       it if the checker is called before the delivery. We will need to make sure
				// 	//       the message is delivered before triggering rule checking.
				localMessage := datatype.NewMessage(
					string(datatype.EventPluginLastExecution),
					pluginName,
					time.Now().UnixNano(),
					map[string]string{},
				)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(localMessage, "node")

				message2 := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusComplete).
					AddPluginRuntimeMeta(*pr).
					AddPodMeta(pod).
					AddPluginMeta(pr.Plugin).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message2.ToWaggleMessage(), "all")
				defer ns.ResourceManager.TerminatePod(pod.Name)
			}
		case v1.PodFailed:
			// one of the containers failed
			// if plugin-controller container failed, we consider the Pod as success

			// Note: The PodFailed event can be received multiple times from Kubernetes.
			//       Yongho checked the content of the events and they look the same.
			//       Thus, we ignore duplicated events.
			if !pr.IsState(datatype.Failed) {
				logger.Info.Printf("Plugin %q failed", pod.Name)
				pr.Failed()
				messageBuilder, err := ns.ResourceManager.AnalyzeFailureOfPod(pod)
				if err != nil {
					logger.Error.Println(err.Error())
				}
				message := messageBuilder.AddPluginRuntimeMeta(*pr).
					AddPluginMeta(pr.Plugin).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message.ToWaggleMessage(), "all")
				defer ns.ResourceManager.TerminatePod(pod.Name)
			}
		case v1.PodUnknown:
			fallthrough
		default:
			// unknown Pod phase; we should notify us in case we
			// care this event
			logger.Error.Printf("plugin %q Pod is in unknown state: %s", pod.Name, pod.Status.Phase)
		}
	case KubernetesEventTypeDeleted:
		logger.Info.Printf("Plugin %q removed", pod.Name)
		var privateMessage datatype.SchedulerEvent
		switch pr.Status.State {
		case datatype.Completed:
			// The Pod is deleted as plugin execution terminated successfully.
			// We do nothing on this transition
			privateMessage = datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusComplete).
				AddReason(fmt.Sprintf("plugin %q successfully removed", pod.Name)).
				Build().(datatype.SchedulerEvent)
		case datatype.Failed:
			// The pod failed and should have been already reported. we do nothing
			privateMessage = datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
				AddReason(fmt.Sprintf("plugin %q removed due to a failure", pod.Name)).
				Build().(datatype.SchedulerEvent)
		default:
			// The pod was deleted for unknown reason. One of the reasons might be
			// that the Pod was deleted from external, e.g. kubectl delete pod.
			// We mark this as a failure.
			pr.Failed()
			privateMessage = datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
				AddReason(fmt.Sprintf("plugin %q deleted from external", pod.Name)).
				Build().(datatype.SchedulerEvent)

			message := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
				AddReason("plugin deleted from external").
				AddPluginRuntimeMeta(*pr).
				AddPluginMeta(pr.Plugin).
				AddPodMeta(pod).
				Build().(datatype.SchedulerEvent)
			ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message.ToWaggleMessage(), "all")
		}
		// trigger the scheduler to schedule next Plugins
		ns.scheduledPlugins.Pop(pr)
		pr.Inactive()
		ns.chanNeedScheduling <- privateMessage
	}
}

// handleKubernetesEventEvent processes Event messages sent from Kubernetes.
// When starting, Kubernetes Informer sends all events from any existing resources.
//
// TODO: As of now, we are ignoring those events as we clean up all resources when
// the scheduler starts. If we want to keep the objects/resources regardless of
// scheduler runs, we may want to leaverage this to construct current state of Plugins
// and resume managing them, instead of killing all of them due to a scheduler restart.
func (ns *NodeScheduler) handleKubernetesEventEvent(e KubernetesEvent) {
	event := e.Event
	logger.Debug.Printf("event %s: %s, %s", event.Name, event.Reason, event.Message)
	logger.Debug.Printf("%v", event)

	// we assume Events are always Add type
	// switch e.Action {
	// case KubernetesEventTypeAdd, KubernetesEventTypeModified:
	// }
	obj := event.InvolvedObject
	switch obj.Kind {
	case "Pod":
		if pr := ns.GoalManager.GetPluginRuntimeByPodUID(string(obj.UID)); pr != nil {
			// There are events on failures that can lead the plugin-controller container
			// to hang, which causes the Pod hanging as well. We consider this as a failure so we remove
			// the Pod. Users should treat this message as a failure of their plugin.
			switch event.Reason {
			case "FailedPostStartHook", "Failed", "FailedMount", "FailedCreatePodSandBox":
				// NOTE: There can be multiple Reasons of a failure. We try to capture them
				//       as much as possible.
				logger.Info.Printf("Plugin %q failed due to %s", obj.Name, event.Reason)
				pr.Failed()
				message := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
					AddPluginRuntimeMeta(*pr).
					AddPluginMeta(pr.Plugin).
					AddReason(event.Reason).
					AddEntry("message", event.Message).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message.ToWaggleMessage(), "all")
				defer ns.ResourceManager.TerminatePod(obj.Name)
			default:
			}
			message := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusEvent).
				AddPluginRuntimeMeta(*pr).
				AddPluginMeta(pr.Plugin).
				AddReason(event.Reason).
				AddEntry("message", event.Message).
				Build().(datatype.SchedulerEvent)
			ns.LogToBeehive.SendWaggleMessageOnNodeAsync(message.ToWaggleMessage(), "all")
		}
	default:
	}
}

func (ns *NodeScheduler) handleKubernetesConfigMapEvent(e KubernetesEvent) {
	cm := e.ConfigMap
	// Currently we only care the ConfigMap that contains a given job (goal) set
	// under the name "waggle-plugin-scheduler-goals"
	if cm.Name != "waggle-plugin-scheduler-goals" {
		return
	}
	switch e.Action {
	case KubernetesEventTypeAdd, KubernetesEventTypeModified:
		logger.Debug.Printf("A bulk goal is received: %v", cm.Data)
		if data, found := cm.Data["goals"]; found {
			var goals []datatype.ScienceGoal
			err := json.Unmarshal([]byte(data), &goals)
			if err != nil {
				logger.Error.Printf("Failed to load bulk goals %s: %q", data, err.Error())
			} else {
				ns.handleBulkGoals(goals)
			}
		} else {
			logger.Error.Printf("Kubernetes ConfigMap %q for goals does not have \"goals\" data: %s",
				cm.Name,
				cm.Data)
		}
	default:
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
			// make sure plugins has its job and goal IDs
			_p.GoalID = goal.ID
			_p.JobID = goal.JobID

			pr := datatype.NewPluginRuntime(_p)
			ns.GoalManager.AddPluginRuntime(pr)
			logger.Debug.Printf("plugin %s is added to the watiting queue", p.Name)
		}
	}
}

func (ns *NodeScheduler) cleanUpGoal(goal *datatype.ScienceGoal) {
	ns.Knowledgebase.DropRules(goal.Name)
	if mySubGoal := goal.GetMySubGoal(ns.NodeID); mySubGoal != nil {
		for _, p := range goal.GetMySubGoal(ns.NodeID).GetPlugins() {
			if pr := ns.GoalManager.GetPluginRuntime(PluginIndex{
				name:   p.Name,
				goalID: goal.ID,
				jobID:  goal.JobID,
			}); pr == nil {
				logger.Error.Printf("failed to remove plugin: plugin name %q for goal %q not registered", p.Name, goal.ID)
				// TODO: we may want to verify what exist and why this happens
			} else {
				if a := ns.readyQueue.Pop(pr); a != nil {
					logger.Debug.Printf("plugin %s is removed from the ready queue", p.Name)
				}
				if a := ns.scheduledPlugins.Pop(pr); a != nil {
					// Pods have their job ID in the name
					var podName string
					if a.Plugin.JobID != "" {
						podName = fmt.Sprintf("%s-%s", a.Plugin.Name, a.Plugin.JobID)
					} else {
						podName = a.Plugin.Name
					}
					if pod, err := ns.ResourceManager.GetPod(podName); err != nil {
						logger.Error.Printf("Failed to get pod of the plugin %q", a.Plugin.Name)
					} else {
						e := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusFailed).
							AddPluginRuntimeMeta(*pr).
							AddPluginMeta(a.Plugin).
							AddPodMeta(pod).
							AddReason("Cleaning up the plugin due to deletion of the goal").
							Build().(datatype.SchedulerEvent)
						ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
						ns.ResourceManager.TerminatePod(podName)
						logger.Info.Printf("plugin %s is removed from running", p.Name)
					}
					ns.GoalManager.DropPluginRuntime(PluginIndex{
						name:   p.Name,
						goalID: goal.ID,
						jobID:  goal.JobID,
					})
				}
			}
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
				e := datatype.NewSchedulerEventBuilder(datatype.EventGoalStatusUpdated).
					AddGoal(&goal).
					Build().(datatype.SchedulerEvent)
				ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
			}
		} else {
			logger.Info.Printf("Adding the new goal %s %q", goal.Name, goal.ID)
			ns.registerGoal(&goal)
			e := datatype.NewSchedulerEventBuilder(datatype.EventGoalStatusReceived).
				AddGoal(&goal).
				Build().(datatype.SchedulerEvent)
			ns.LogToBeehive.SendWaggleMessageOnNodeAsync(e.ToWaggleMessage(), "all")
		}
	}
	// Remove any existing goal that is not included in the new goal set
	for _, goal := range ns.GoalManager.ScienceGoals {
		if _, exist := goalsToKeep[goal.ID]; !exist {
			ns.cleanUpGoal(&goal)
			event := datatype.NewSchedulerEventBuilder(datatype.EventGoalStatusRemoved).
				AddGoal(&goal).
				Build().(datatype.SchedulerEvent)
			ns.LogToBeehive.SendWaggleMessageOnNodeAsync(event.ToWaggleMessage(), "all")
		}
	}
}
