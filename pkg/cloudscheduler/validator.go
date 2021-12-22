package cloudscheduler

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
)

type JobValidator struct {
}

func NewJobValidator() (*JobValidator, error) {
	return &JobValidator{}, nil
}

// ValidateJobAndCreateScienceGoal validates user job and returns a science goals
// created from the job. It also returns a list of errors in validation if any
func (jv *JobValidator) ValidateJobAndCreateScienceGoal(job *datatype.Job, meta *MetaHandler) (scienceGoal *datatype.ScienceGoal, errorList []error) {
	logger.Info.Printf("Validating %s", job.Name)
	scienceGoal = new(datatype.ScienceGoal)
	scienceGoal.ID = job.ID
	scienceGoal.Name = job.Name

	for _, n := range job.Nodes {
		node := n
		var subGoal datatype.SubGoal
		for _, p := range job.Plugins {
			plugin := p
			// Check 1: plugin exists in ECR
			// exists := pluginExists(plugin)
			// if !exists {
			// 	errorList = append(errorList, fmt.Errorf("%s:%s not exist in ECR", plugin.Name, plugin.Version))
			// 	continue
			// }
			// logger.Info.Printf("%s:%s exists in ECR", plugin.Name, plugin.Version)

			// Check 2: node supports hardware requirements of the plugin
			supported, unsupportedHardwareList := node.GetPluginHardwareUnsupportedList(plugin)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s:%s required hardware not supported by %s: %v", plugin.Name, plugin.PluginSpec.Version, node.Name, unsupportedHardwareList))
				continue
			}
			logger.Info.Printf("%s:%s hardware %v supported by %s", plugin.Name, plugin.PluginSpec.Version, plugin.Hardware, node.Name)

			// Check 3: architecture of the plugin is supported by node
			supported, supportedDevices := node.GetPluginArchitectureSupportedDevices(plugin)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s:%s architecture not supported by %s", plugin.Name, plugin.PluginSpec.Version, node.Name))
				continue
			}
			logger.Info.Printf("%s:%s architecture %v supported by %v of node %s", plugin.Name, plugin.PluginSpec.Version, plugin.Architecture, supportedDevices, node.Name)

			// Check 4: the required resource is available in node devices
			for _, device := range supportedDevices {
				supported, profiles := device.GetUnsupportedPluginProfiles(plugin)
				if !supported {
					errorList = append(errorList, fmt.Errorf("%s:%s required resource not supported by device %s of node %s", plugin.Name, plugin.PluginSpec.Version, device.Name, node.Name))
					continue
				}
				// Filter out unsupported knob settings
				for _, profile := range profiles {
					err := plugin.RemoveProfile(profile)
					if err != nil {
						logger.Error.Printf("%s", err)
					}
				}
			}
			subGoal.Plugins = append(subGoal.Plugins, &plugin)
		}
		// Check 4: conditions of job are valid

		// Check 5: valiables are valid
		if len(subGoal.Plugins) > 0 {
			subGoal.Node = &node
			subGoal.Sciencerules = job.ScienceRules
			scienceGoal.SubGoals = append(scienceGoal.SubGoals, &subGoal)
		}
	}
	//
	// for _, plugin := range job.Plugins {
	// 	// Check 1: plugin existence in ECR
	// 	exists := pluginExists(plugin)
	// 	if !exists {
	// 		errorList = append(errorList, fmt.Errorf("%s:%s not exist in ECR", plugin.Name, plugin.Version))
	// 		continue
	// 	}
	// 	logger.Info.Printf("%s:%s exists in ECR", plugin.Name, plugin.Version)
	//
	// 	// Check 2: plugins run on target nodes and supported by node hardware and resource
	// 	for _, node := range job.Nodes {
	// 		supported, supportedDevices := node.GetPluginArchitectureSupportedDevices(plugin)
	// 		if !supported {
	// 			errorList = append(errorList, fmt.Errorf("%s:%s architecture not supported by %s", plugin.Name, plugin.Version, node.Name))
	// 			continue
	// 		}
	// 		logger.Info.Printf("%s:%s architecture %v supported by %v of node %s", plugin.Name, plugin.Version, plugin.Architecture, supportedDevices, node.Name)
	//
	// 		supported, unsupportedHardwareList := node.GetPluginHardwareUnsupportedList(plugin)
	// 		if !supported {
	// 			errorList = append(errorList, fmt.Errorf("%s:%s required hardware not supported by %s: %v", plugin.Name, plugin.Version, node.Name, unsupportedHardwareList))
	// 			continue
	// 		}
	// 		logger.Info.Printf("%s:%s hardware %v supported by %s", plugin.Name, plugin.Version, plugin.Hardware, node.Name)
	//
	// 		for _, device := range supportedDevices {
	// 			profiles := device.GetUnsupportedPluginProfiles(plugin)
	// 			logger.Info.Printf("hi")
	// 			logger.Info.Printf("%v", profiles)
	// 			// if !supported {
	// 			// 	errorList = append(errorList, fmt.Errorf(
	// 			// 		"%s:%s not enough resources to be run on %s device of %s node",
	// 			// 		plugin.Name,
	// 			// 		plugin.Version,
	// 			// 		device.Name,
	// 			// 		node.Name,
	// 			// 	))
	// 			// 	continue
	// 			// }
	// 			// Remove profiles
	// 			for _, profile := range profiles {
	// 				err := plugin.RemoveProfile(profile)
	// 				if err != nil {
	// 					ErrorLogger.Printf("%s", err)
	// 				}
	// 			}
	//
	// 			logger.Info.Printf("%v\n", plugin)
	// 		}
	//
	// 		// Check 3: if the profiles satisfy the minimum performance requirement of job
	// 	}

	// Check 4: conditions of job are valid

	// Check 5: valiables are valid

	// }
	return
}
