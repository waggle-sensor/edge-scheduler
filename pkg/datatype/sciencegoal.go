package datatype

import "fmt"

// ScienceGoal structs local goals and success criteria
type ScienceGoal struct {
	ID         string     `yaml:"id"`
	Name       string     `yaml:"name,omitempty"`
	SubGoals   []*SubGoal `yaml:"subgoals,omitempty"`
	Conditions []string   `yaml:"conditions,omitempty"`
}

// GetMySubGoal returns the subgoal assigned to node
func (g *ScienceGoal) GetMySubGoal(nodeName string) *SubGoal {
	for _, subGoal := range g.SubGoals {
		if subGoal.Node.Name == nodeName {
			return subGoal
		}
	}
	return nil
}

// SubGoal structs node-specific goal along with conditions and rules
type SubGoal struct {
	Node         *Node     `yaml:"node,omitempty"`
	Plugins      []*Plugin `yaml:"plugins,omitempty"`
	Sciencerules []string  `yaml:"sciencerules,omitempty"`
}

// UpdatePluginContext updates plugin's context event within the subgoal
// It returns an error if it fails to update context status of the plugin
func (sg *SubGoal) UpdatePluginContext(contextEvent EventPluginContext) error {
	for _, plugin := range sg.Plugins {
		if plugin.Name == contextEvent.PluginName {
			return plugin.UpdatePluginContext(contextEvent.Status)
		}
	}
	return fmt.Errorf("failed to update context (%s) of plugin %s", contextEvent.Status, contextEvent.PluginName)
}

// GetSchedulablePlugins returns a list of plugins that are schedulable.
// A plugin is schedulable when its ContextStatus is Runnable and
// SchedulingStatus is not Running
func (sg *SubGoal) GetSchedulablePlugins() (schedulable []*Plugin) {
	for _, plugin := range sg.Plugins {
		if plugin.Status.ContextStatus == Runnable &&
			plugin.Status.SchedulingStatus != Running {
			schedulable = append(schedulable, plugin)
		}
	}
	return
}

// GetPlugin returns the plugin that matches with given pluginName
func (sg *SubGoal) GetPlugin(pluginName string) *Plugin {
	for _, plugin := range sg.Plugins {
		if plugin.Name == pluginName {
			return plugin
		}
	}
	return nil
}

type Goal struct {
	Name    string `yaml:"name"`
	Plugins []*PluginSpec
}
