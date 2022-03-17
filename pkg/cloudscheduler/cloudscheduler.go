package cloudscheduler

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
)

const maxChannelBuffer = 100

// CloudScheduler structs the cloud scheduler
type CloudScheduler struct {
	Name                string
	Version             string
	GoalManager         *CloudGoalManager
	Validator           *JobValidator
	APIServer           *APIServer
	chanFromGoalManager chan datatype.Event
}

func (cs *CloudScheduler) ValidateJobAndCreateScienceGoal(jobID string) (errorList []error) {
	job, err := cs.GoalManager.GetJob(jobID)
	if err != nil {
		return []error{err}
	}
	scienceGoalBuilder := datatype.NewScienceGoalBuilder(job.Name, job.JobID)
	logger.Info.Printf("Validating %s...", job.Name)
	// Step 1: Resolve node tags
	job.AddNodes(cs.Validator.GetNodeNamesByTags(job.NodeTags))
	if len(job.Nodes) < 1 {
		return []error{fmt.Errorf("Node is not selected")}
	}
	for nodeName := range job.Nodes {
		approvedPlugins := []*datatype.Plugin{}
		nodeManifest := cs.Validator.GetNodeManifest(nodeName)
		if nodeManifest == nil {
			errorList = append(errorList, fmt.Errorf("%s does not exist", nodeName))
			continue
		}
		for _, plugin := range job.Plugins {
			pluginManifest := cs.Validator.GetPluginManifest(plugin)
			if pluginManifest == nil {
				errorList = append(errorList, fmt.Errorf("%s does not exist in ECR", plugin.PluginSpec.Image))
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
			supported, unsupportedHardwareList := nodeManifest.GetPluginHardwareUnsupportedList(pluginManifest)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s does not support hardware %v required by %s (%s)", nodeName, unsupportedHardwareList, plugin.Name, plugin.PluginSpec.Image))
				continue
			}
			logger.Info.Printf("%s passed Check 2", plugin.Name)

			// Check 3: architecture of the plugin is supported by node
			supported, supportedDevices := nodeManifest.GetPluginArchitectureSupportedDevices(pluginManifest)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s does not support architecture %v required by %s (%s)", nodeName, pluginManifest.Architecture, plugin.Name, plugin.PluginSpec.Image))
				continue
			}
			logger.Info.Printf("%s passed Check 3", plugin.Name)

			// Check 4: the required resource is available in node devices
			for _, device := range supportedDevices {
				supported, _ := device.GetUnsupportedPluginProfiles(pluginManifest)
				if !supported {
					errorList = append(errorList, fmt.Errorf("%s (%s) does not support resource required by %s (%s)", nodeName, device.Name, plugin.Name, plugin.PluginSpec.Image))
					continue
				}
				// // Filter out unsupported knob settings
				// for _, profile := range profiles {
				// 	err := plugin.RemoveProfile(profile)
				// 	if err != nil {
				// 		logger.Error.Printf("%s", err)
				// 	}
				// }
			}
			approvedPlugins = append(approvedPlugins, plugin)
		}
		// Check 4: conditions of job are valid

		// Check 5: valiables are valid
		if len(approvedPlugins) > 0 {
			scienceGoalBuilder = scienceGoalBuilder.AddSubGoal(nodeName, approvedPlugins, job.ScienceRules)
		}
	}
	if len(errorList) > 0 {
		logger.Info.Printf("Validation failed for Job ID %q: %v", jobID, errorList)
		return errorList
	} else {
		logger.Info.Printf("Updating science goal for JOB ID %q", jobID)
		job.ScienceGoal = scienceGoalBuilder.Build()
		job.UpdateStatus(datatype.JobSubmitted)
		cs.GoalManager.UpdateJob(job, true)
		return nil
	}
}

func (cs *CloudScheduler) Run() {
	go cs.APIServer.Run()
	logger.Info.Printf("Cloud Scheduler %s starts...", cs.Name)
	for {
		select {
		case event := <-cs.chanFromGoalManager:
			logger.Debug.Printf("%s: %q", event.ToString(), event.GetGoalName())
			switch event.Type {
			case datatype.EventGoalStatusNew:
				scienceGoal, err := cs.GoalManager.GetScienceGoal(event.GetGoalID())
				if err != nil {
					logger.Error.Printf("Failed to get science goal %q", event.GetGoalID())
					break
				}
				logger.Info.Printf("New goal %q is submitted for job id %q.", scienceGoal.Name, scienceGoal.JobID)
			}
		}
	}
}
