package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var (
	chanToValidator chan *Job
	nodes           []Node
	plugins         []Plugin
)

// InitializeValidator initializes the validator
func InitializeValidator() {
	nodes, _ = getNodesFromDirectory()
	// InfoLogger.Printf("%v", nodes)
	plugins, _ = getPluginsFromDirectory()
	// InfoLogger.Printf("%v", plugins)
	chanToValidator = make(chan *Job)
}

// ValidateJobAndCreateScienceGoal validates user job and returns a science goals
// created from the job. It also returns a list of errors in validation if any
func ValidateJobAndCreateScienceGoal(job *Job) (scienceGoal *ScienceGoal, errorList []error) {
	InfoLogger.Printf("Validating %s...", job.Name)
	scienceGoal = new(ScienceGoal)
	scienceGoal.ID = job.ID
	scienceGoal.Name = job.Name

	for _, node := range job.Nodes {
		var subGoal SubGoal
		for _, plugin := range job.Plugins {
			// Check 1: plugin exists in ECR
			exists := pluginExists(plugin)
			if !exists {
				errorList = append(errorList, fmt.Errorf("%s:%s not exist in ECR", plugin.Name, plugin.Version))
				continue
			}
			InfoLogger.Printf("%s:%s exists in ECR", plugin.Name, plugin.Version)

			// Check 2: node supports hardware requirements of the plugin
			supported, unsupportedHardwareList := node.GetPluginHardwareUnsupportedList(plugin)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s:%s required hardware not supported by %s: %v", plugin.Name, plugin.Version, node.Name, unsupportedHardwareList))
				continue
			}
			InfoLogger.Printf("%s:%s hardware %v supported by %s", plugin.Name, plugin.Version, plugin.Hardware, node.Name)

			// Check 3: architecture of the plugin is supported by node
			supported, supportedDevices := node.GetPluginArchitectureSupportedDevices(plugin)
			if !supported {
				errorList = append(errorList, fmt.Errorf("%s:%s architecture not supported by %s", plugin.Name, plugin.Version, node.Name))
				continue
			}
			InfoLogger.Printf("%s:%s architecture %v supported by %v of node %s", plugin.Name, plugin.Version, plugin.Architecture, supportedDevices, node.Name)

			// Check 4: the required resource is available in node devices
			for _, device := range supportedDevices {
				supported, profiles := device.GetUnsupportedPluginProfiles(plugin)
				if !supported {
					errorList = append(errorList, fmt.Errorf("%s:%s required resource not supported by device %s of node %s", plugin.Name, plugin.Version, device.Name, node.Name))
					continue
				}
				// Filter out unsupported knob settings
				for _, profile := range profiles {
					err := plugin.RemoveProfile(profile)
					if err != nil {
						ErrorLogger.Printf("%s", err)
					}
				}
			}
			subGoal.Plugins = append(subGoal.Plugins, plugin)
		}
		// Check 4: conditions of job are valid

		// Check 5: valiables are valid
		if len(subGoal.Plugins) > 0 {
			subGoal.Node = node
			subGoal.Conditions = job.Conditions
			subGoal.Rules = job.Rules
			scienceGoal.SubGoals = append(scienceGoal.SubGoals, subGoal)
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
	// 	InfoLogger.Printf("%s:%s exists in ECR", plugin.Name, plugin.Version)
	//
	// 	// Check 2: plugins run on target nodes and supported by node hardware and resource
	// 	for _, node := range job.Nodes {
	// 		supported, supportedDevices := node.GetPluginArchitectureSupportedDevices(plugin)
	// 		if !supported {
	// 			errorList = append(errorList, fmt.Errorf("%s:%s architecture not supported by %s", plugin.Name, plugin.Version, node.Name))
	// 			continue
	// 		}
	// 		InfoLogger.Printf("%s:%s architecture %v supported by %v of node %s", plugin.Name, plugin.Version, plugin.Architecture, supportedDevices, node.Name)
	//
	// 		supported, unsupportedHardwareList := node.GetPluginHardwareUnsupportedList(plugin)
	// 		if !supported {
	// 			errorList = append(errorList, fmt.Errorf("%s:%s required hardware not supported by %s: %v", plugin.Name, plugin.Version, node.Name, unsupportedHardwareList))
	// 			continue
	// 		}
	// 		InfoLogger.Printf("%s:%s hardware %v supported by %s", plugin.Name, plugin.Version, plugin.Hardware, node.Name)
	//
	// 		for _, device := range supportedDevices {
	// 			profiles := device.GetUnsupportedPluginProfiles(plugin)
	// 			InfoLogger.Printf("hi")
	// 			InfoLogger.Printf("%v", profiles)
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
	// 			InfoLogger.Printf("%v\n", plugin)
	// 		}
	//
	// 		// Check 3: if the profiles satisfy the minimum performance requirement of job
	// 	}

	// Check 4: conditions of job are valid

	// Check 5: valiables are valid

	// }
	return
}

// RunValidator is a goroutine that validates job requests and builds science goals
func RunValidator() {
	for {
		job := <-chanToValidator

		// TODO: Add error hanlding here
		scienceGoal, errorList := ValidateJobAndCreateScienceGoal(job)
		if errorList != nil {
			for _, err := range errorList {
				ErrorLogger.Printf("%s", err)
			}
		}
		InfoLogger.Printf("%+v\n", scienceGoal)
		chanToJobManager <- scienceGoal
	}
}

func getNodesFromDirectory() (nodes []Node, err error) {
	nodeFiles, _ := filepath.Glob("./data/nodes/*.yaml")
	for _, filePath := range nodeFiles {
		dat, _ := ioutil.ReadFile(filePath)
		var node Node
		_ = yaml.Unmarshal(dat, &node)
		nodes = append(nodes, node)
	}
	return
}

func getPluginsFromDirectory() (plugins []Plugin, err error) {
	nodeFiles, _ := filepath.Glob("./data/plugins/*.yaml")
	for _, filePath := range nodeFiles {
		dat, _ := ioutil.ReadFile(filePath)
		var plugin Plugin
		_ = yaml.Unmarshal(dat, &plugin)
		plugins = append(plugins, plugin)
	}
	return
}

func pluginExists(plugin Plugin) bool {
	return pluginExistInArray(plugin, plugins)
}

func pluginExistInArray(plugin Plugin, plugins []Plugin) bool {
	for _, pluginInArray := range plugins {
		if pluginInArray.Name == plugin.Name &&
			pluginInArray.Version == plugin.Version {
			return true
		}
	}
	return false
}

func nodeExistInArray(node Node, nodes []Node) bool {
	for _, nodeInarray := range nodes {
		if nodeInarray.Name == node.Name {
			return true
		}
	}
	return false
}

func getPlugin(name string, version string) (Plugin, error) {
	for _, plugin := range plugins {
		if name == plugin.Name && version == plugin.Version {
			return plugin, nil
		}
	}
	return Plugin{}, fmt.Errorf("Plugin %s:%s not exist in ECR", name, version)
}

func getPluginsByTags(tags []string) (pluginsFound []Plugin) {
	for _, plugin := range plugins {
		for _, tag := range tags {
			for _, pluginTag := range plugin.Tags {
				if tag == pluginTag {
					exists := pluginExistInArray(plugin, pluginsFound)
					if !exists {
						pluginsFound = append(pluginsFound, plugin)
					}
					break
				}
			}
		}
	}
	return
}

func getNodesByTags(tags []string) (nodesFound []Node) {
	for _, node := range nodes {
		for _, tag := range tags {
			for _, nodeTag := range node.Tags {
				if tag == nodeTag {
					exists := nodeExistInArray(node, nodesFound)
					if !exists {
						nodesFound = append(nodesFound, node)
					}
					break
				}
			}
		}
	}
	return
}

func getNode(name string) (Node, error) {
	for _, node := range nodes {
		if name == node.Name {
			return node, nil
		}
	}
	return Node{}, fmt.Errorf("Node %s not exist in the system", name)
}

// func main() {
// 	InitializeValidator()
//
// 	dat, _ := ioutil.ReadFile("./data/jobs/job1_image_collection.yaml")
// 	var job Job
// 	_ = yaml.Unmarshal(dat, &job)
//
// 	foundPlugins := getPluginsByTags(job.PluginTags)
// 	job.Plugins = foundPlugins
//
// 	foundNodes := getNodesByTags(job.NodeTags)
// 	job.Nodes = foundNodes
//
// 	InfoLogger.Printf("%v", job)
//
// 	// jobName := "Example Job"
// 	//
// 	// // Use plugin and node tags to specify nodes and plugins
// 	// job := createExampleJob(jobName)
// 	// Or specify them explicitly
// 	// var job *Job
// 	// job = new(Job)
// 	// cloudCoverPlugin, _ := getPlugin("plugin-cloudcover", "0.1.0")
// 	// wb01Node, _ := getNode("wb01")
// 	// job.Plugins = append(job.Plugins, cloudCoverPlugin)
// 	// job.Nodes = append(job.Nodes, wb01Node)
//
// 	scienceGoal, errorList := ValidateJobAndCreateScienceGoal(&job)
// 	if errorList != nil {
// 		for _, err := range errorList {
// 			ErrorLogger.Printf("%s", err)
// 		}
// 	} else {
// 		InfoLogger.Printf("%+v\n", scienceGoal)
// 		dat, _ := yaml.Marshal(scienceGoal)
// 		ioutil.WriteFile("sciencegoal.yaml", dat, 0644)
//
// 	}
// }
