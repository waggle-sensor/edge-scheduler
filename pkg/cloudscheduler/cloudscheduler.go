package cloudscheduler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

const maxChannelBuffer = 100

// CloudScheduler structs the cloud scheduler
type CloudScheduler struct {
	Name                string
	Version             string
	Config              *CloudSchedulerConfig
	GoalManager         *CloudGoalManager
	Validator           *JobValidator
	APIServer           *APIServer
	chanFromGoalManager chan datatype.Event
	MetricsCollector    *prometheus.Collector
	eventListener       *interfacing.RabbitMQHandler
}

func (cs *CloudScheduler) Configure() error {
	// Loading job database
	if err := cs.GoalManager.OpenJobDB(); err != nil {
		return err
	}
	// Loading science goals from the job database
	// to bring the latest status of jobs to the scheduler
	if err := cs.GoalManager.LoadScienceGoalsFromJobDB(); err != nil {
		return err
	}
	// Loading node and plugin database
	if err := cs.Validator.LoadDatabase(); err != nil {
		return err
	}
	// loading plugin whitelist
	cs.Validator.LoadPluginWhitelist()
	// Setting up Prometheus metrics
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(NewMetricsCollector(cs))
	cs.APIServer.ConfigureAPIs(reg)

	// Setting up RabbitMQ connection to receive scheduling events from nodes
	if !cs.Config.NoRabbitMQ {
		logger.Info.Printf(
			"Using RabbitMQ at %s with user %s",
			cs.Config.RabbitmqURL,
			cs.Config.RabbitmqUsername,
		)
		cs.eventListener = interfacing.NewRabbitMQHandler(
			cs.Config.RabbitmqURL,
			cs.Config.RabbitmqUsername,
			cs.Config.RabbitmqPassword,
			cs.Config.RabbitmqCaCertPath,
			"",
		)
	}
	return nil
}

func (cs *CloudScheduler) ValidateJobAndCreateScienceGoal(jobID string, user *User, dryrun bool) (errorList []error) {
	job, err := cs.GoalManager.GetJob(jobID)
	if err != nil {
		return []error{err}
	}
	scienceGoalBuilder := datatype.NewScienceGoalBuilder(job.Name, job.JobID)
	logger.Info.Printf("Validating %s...", job.Name)
	// Step 1: Resolve node tags
	job.AddNodes(cs.Validator.GetNodeNamesByTags(job.NodeTags))
	// TODO: Jobs may be submitted without nodes in the future
	//       For example, Chicago nodes without having any node in Chicago yet
	if len(job.Nodes) < 1 {
		return []error{fmt.Errorf("Node is not selected")}
	}
	// Check if email is set for notification
	if len(job.NotificationOn) > 0 {
		if job.Email == "" {
			return []error{fmt.Errorf("No email is set for notification")}
		}
		// Check if given notification types are valid
		for _, s := range job.NotificationOn {
			switch s {
			case datatype.JobCreated,
				datatype.JobDrafted,
				datatype.JobSubmitted,
				datatype.JobRunning,
				datatype.JobComplete,
				datatype.JobSuspended,
				datatype.JobRemoved:
				continue
			default:
				errorList = append(errorList, fmt.Errorf("No type %q in Job notification", s))
			}
		}
		if len(errorList) > 1 {
			return
		}
	}
	for nodeName := range job.Nodes {
		// Check 0: if the user can schedule
		ret, err := user.CanScheduleOnNode(nodeName)
		if err != nil {
			errorList = append(errorList, err)
			continue
		} else if ret == false {
			errorList = append(errorList, fmt.Errorf("User %s does not have permission for node %s", user.GetUserName(), nodeName))
			continue
		}
		approvedPlugins := []*datatype.Plugin{}
		nodeManifest := cs.Validator.GetNodeManifest(nodeName)
		if nodeManifest == nil {
			errorList = append(errorList, fmt.Errorf("%s does not exist", nodeName))
			continue
		}
		for _, plugin := range job.Plugins {
			pluginImage, err := plugin.GetPluginImage()
			if err != nil {
				errorList = append(errorList, fmt.Errorf("%s does not specify plugin image", plugin.Name))
				continue
			}
			pluginManifest := cs.Validator.GetPluginManifest(pluginImage, true)
			if pluginManifest == nil {
				// we also check if the image is in the whitelist. If so, we approve for the plugin
				if cs.Validator.IsPluginWhitelisted(pluginImage) {
					approvedPlugins = append(approvedPlugins, plugin)
				} else {
					errorList = append(errorList, fmt.Errorf("%s does not exist in ECR", plugin.PluginSpec.Image))
				}
				continue
			}
			// Check 1: plugin exists in ECR
			// exists := pluginExists(plugin)
			// if !exists {
			// 	errorList = append(errorList, fmt.Errorf("%s:%s not exist in ECR", plugin.Name, plugin.Version))
			// 	continue
			// }
			// logger.Info.Printf("%s:%s exists in ECR", plugin.Name, plugin.Version)

			// Check 2: node supports hardware requirements of the plugin
			// TODO: plugin manifest does not yet have sensor requirement
			// supported, unsupportedHardwareList := nodeManifest.GetUnsupportedListOfPluginSensors(pluginManifest)
			// if !supported {
			// 	errorList = append(errorList, fmt.Errorf("%s does not support hardware %v required by %s (%s)", nodeName, unsupportedHardwareList, plugin.Name, plugin.PluginSpec.Image))
			// 	continue
			// }
			// logger.Info.Printf("%s passed Check 2", plugin.Name)

			// Check 3: architecture of the plugin is supported by node
			supported, _ := nodeManifest.GetPluginArchitectureSupportedComputes(pluginManifest)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s does not support architecture %v required by %s (%s)", nodeName, pluginManifest.GetArchitectures(), plugin.Name, plugin.PluginSpec.Image))
				continue
			}
			logger.Info.Printf("%s passed Check 3", plugin.Name)

			// Check 4: the required resource is available in node devices
			// for _, c := range supportedComputes {
			// 	supported, _ := c.GetUnsupportedPluginProfiles(pluginManifest)
			// 	if !supported {
			// 		errorList = append(errorList, fmt.Errorf("%s (%s) does not support resource required by %s (%s)", nodeName, c.Name, plugin.Name, plugin.PluginSpec.Image))
			// 		continue
			// 	}
			// // Filter out unsupported knob settings
			// for _, profile := range profiles {
			// 	err := plugin.RemoveProfile(profile)
			// 	if err != nil {
			// 		logger.Error.Printf("%s", err)
			// 	}
			// }
			// }
			approvedPlugins = append(approvedPlugins, plugin)
		}
		// Check 4: conditions of job are valid

		// Check 5: valiables are valid
		var rules []*datatype.ScienceRule
		for _, rule := range job.ScienceRules {
			r, err := datatype.NewScienceRule(rule)
			if err != nil {
				errorList = append(errorList,
					fmt.Errorf("Failed to parse science rule %q: %s", rule, err.Error()))
				continue
			}
			rules = append(rules, r)
		}
		scienceGoalBuilder = scienceGoalBuilder.AddSubGoal(nodeName, approvedPlugins, rules)
	}
	if len(errorList) > 0 {
		logger.Info.Printf("Validation failed for Job ID %q: %v", jobID, errorList)
		return errorList
	}
	logger.Info.Printf("Updating science goal for JOB ID %q", jobID)
	job.ScienceGoal = scienceGoalBuilder.Build()
	if dryrun {
		cs.GoalManager.UpdateJob(job, false)
	} else {
		cs.GoalManager.UpdateJob(job, true)
	}
	return nil
}

func (cs *CloudScheduler) updateNodes(nodes []string) {
	for _, nodeName := range nodes {
		var goals []*datatype.ScienceGoal
		for _, g := range cs.GoalManager.GetScienceGoalsForNode(nodeName) {
			goals = append(goals, g.ShowMyScienceGoal(nodeName))
		}
		// if no science goal is assigned to the node return an empty list []
		// returning null may raise an exception in edge scheduler
		if len(goals) < 1 {
			goals = make([]*datatype.ScienceGoal, 0)
		}
		blob, err := json.MarshalIndent(goals, "", "  ")
		if err != nil {
			logger.Error.Printf("Failed to compress goals for node %q before pushing", nodeName)
		} else {
			event := datatype.NewEventBuilder(datatype.EventGoalStatusUpdated).AddEntry("goals", string(blob)).Build()
			cs.APIServer.Push(nodeName, &event)
		}
	}
}

func (cs *CloudScheduler) Run() {
	logger.Info.Printf("Cloud Scheduler %s starts...", cs.Name)
	go cs.APIServer.Run()
	chanEventFromNode := make(chan *datatype.Event)
	if cs.eventListener != nil {
		cs.eventListener.SubscribeEvents("waggle.msg", "to-scheduler", datatype.EventRabbitMQSubscriptionPatternGoals, chanEventFromNode)
	}
	// Timer for job re-evaluation
	ticker := time.NewTicker(1 * time.Second)
	if cs.Config.JobReevaluationIntervalSecond > 0 {
		ticker = time.NewTicker(time.Duration(cs.Config.JobReevaluationIntervalSecond) * time.Second)
	} else {
		ticker.Stop()
	}
	for {
		select {
		case <-ticker.C:
			logger.Debug.Printf("Job re-evaluation")
		case event := <-chanEventFromNode:
			logger.Debug.Printf("%s:%v", event.ToString(), event)
			// TODO: stat aggregator for jobs may use this event
			sender := event.GetEntry("vsn")
			// sender must be identified
			switch event.Type {
			case datatype.EventGoalStatusReceived, datatype.EventGoalStatusUpdated:
				goalID := event.GetGoalID()
				logger.Debug.Printf("%s received science goal %s", sender, goalID)
				scienceGoal, err := cs.GoalManager.GetScienceGoal(goalID)
				if err != nil {
					logger.Error.Printf("Failed to find science goal %s", goalID)
					break
				}
				job, err := cs.GoalManager.GetJob(scienceGoal.JobID)
				if err != nil {
					logger.Error.Printf("Failed to get job of the science goal %q: %s", goalID, err.Error())
					break
				}
				job.Runs()
				err = cs.GoalManager.UpdateJob(job, false)
				if err != nil {
					logger.Error.Printf("Failed to update status of job %q: %s", scienceGoal.JobID, err.Error())
					break
				}
			}
			// TODO: How do we determine if a job is failed
			//       by looking at EventPluginStatusFailed?
		case event := <-cs.chanFromGoalManager:
			logger.Debug.Printf("%s: %q", event.ToString(), event.GetGoalName())
			switch event.Type {
			case datatype.EventJobStatusRemoved:
				job, err := cs.GoalManager.GetJob(event.GetJobID())
				if err != nil {
					logger.Error.Printf("Failed to get job %q", event.GetJobID())
					break
				}
				// The job is removed. Corresponding science goal should also be removed
				if job.ScienceGoal != nil {
					scienceGoal, err := cs.GoalManager.GetScienceGoal(job.ScienceGoal.ID)
					if err != nil {
						logger.Error.Printf("Failed to get science goal %q", job.ScienceGoal.ID)
						break
					}
					NodesToUpdate := scienceGoal.GetSubjectNodes()
					if err = cs.GoalManager.RemoveScienceGoal(scienceGoal.ID); err != nil {
						logger.Error.Printf("Failed to remove science goal %q", scienceGoal.ID)
						break
					}
					logger.Info.Printf("Goal %q is removed for job %q.", scienceGoal.Name, scienceGoal.JobID)
					cs.updateNodes(NodesToUpdate)
				}
			case datatype.EventJobStatusSuspended:
				job, err := cs.GoalManager.GetJob(event.GetJobID())
				if err != nil {
					logger.Error.Printf("Failed to get job %q", event.GetJobID())
					break
				}
				// The job is removed. Corresponding science goal should also be removed
				if job.ScienceGoal != nil {
					scienceGoal, err := cs.GoalManager.GetScienceGoal(job.ScienceGoal.ID)
					if err != nil {
						logger.Error.Printf("Failed to get science goal %q", job.ScienceGoal.ID)
						break
					}
					NodesToUpdate := scienceGoal.GetSubjectNodes()
					if err = cs.GoalManager.RemoveScienceGoal(scienceGoal.ID); err != nil {
						logger.Error.Printf("Failed to remove science goal %q", scienceGoal.ID)
						break
					}
					logger.Info.Printf("Goal %q is suspended for job %q.", scienceGoal.Name, scienceGoal.JobID)
					cs.updateNodes(NodesToUpdate)
				}
			case datatype.EventGoalStatusSubmitted:
				scienceGoal, err := cs.GoalManager.GetScienceGoal(event.GetGoalID())
				if err != nil {
					logger.Error.Printf("Failed to get science goal %q", event.GetGoalID())
					break
				}
				logger.Info.Printf("Goal %q is submitted for job id %q.", scienceGoal.Name, scienceGoal.JobID)
				NodesToUpdate := scienceGoal.GetSubjectNodes()
				cs.updateNodes(NodesToUpdate)
			}
		}
	}
}
