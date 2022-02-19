package datatype

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	uuid "github.com/nu7hatch/gouuid"
)

type ScienceGoalBuilder struct {
	sg ScienceGoal
}

func NewScienceGoalBuilder(goalName string) *ScienceGoalBuilder {
	id, _ := uuid.NewV4()
	return &ScienceGoalBuilder{
		sg: ScienceGoal{
			ID:   id.String(),
			Name: goalName,
		},
	}
}

func (sgb *ScienceGoalBuilder) AddSubGoal(nodeID string, pluginSpecs []*PluginSpec, scienceRules []string) *ScienceGoalBuilder {
	var subGoal SubGoal
	subGoal.Node = &Node{
		Name: nodeID,
	}
	subGoal.ScienceRules = scienceRules
	for _, pluginSpec := range pluginSpecs {
		subGoal.Plugins = append(subGoal.Plugins, &Plugin{
			Name:       pluginSpec.Name,
			PluginSpec: pluginSpec,
			Status: PluginStatus{
				SchedulingStatus: Waiting,
			},
			GoalID: sgb.sg.ID,
		})
	}
	specjson, err := json.Marshal(subGoal)
	if err != nil {
		// We cannot proceed anymore
		return nil
	}
	sum := sha256.Sum256(specjson)
	// instance := hex.EncodeToString(sum[:])[:8]
	subGoal.checksum = hex.EncodeToString(sum[:])
	sgb.sg.SubGoals = append(sgb.sg.SubGoals, &subGoal)
	return sgb
}

func (sgb *ScienceGoalBuilder) Build() *ScienceGoal {
	return &sgb.sg
}

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
	Name         string    `yaml:"name,omitempty"`
	Node         *Node     `yaml:"node,omitempty"`
	Plugins      []*Plugin `yaml:"plugins,omitempty"`
	ScienceRules []string  `yaml:"sciencerules,omitempty"`
	checksum     string
}

func NewSubGoal(goalID string, nodeID string, plugins []*PluginSpec) *SubGoal {
	var subGoal SubGoal
	subGoal.Node = &Node{
		Name: nodeID,
	}
	for _, pluginSpec := range plugins {
		subGoal.Plugins = append(subGoal.Plugins, &Plugin{
			Name:       pluginSpec.Name,
			PluginSpec: pluginSpec,
			Status: PluginStatus{
				SchedulingStatus: Waiting,
			},
		})
	}
	specjson, err := json.Marshal(subGoal)
	if err != nil {
		return nil
	}
	sum := sha256.Sum256(specjson)
	// instance := hex.EncodeToString(sum[:])[:8]
	subGoal.checksum = hex.EncodeToString(sum[:])
	return &subGoal
}

func (sg *SubGoal) CompareChecksum(otherSubGoal *SubGoal) bool {
	if sg.checksum == otherSubGoal.checksum {
		return true
	} else {
		return false
	}
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

type JobTemplate struct {
	Name         string `yaml:"name"`
	Plugins      []*PluginSpec
	ScienceRules []string `yaml:"science_rules"`
}

func (j *JobTemplate) ConvertJobTemplateToScienceGoal(nodeID string) *ScienceGoal {
	return NewScienceGoalBuilder(j.Name).
		AddSubGoal(nodeID, j.Plugins, j.ScienceRules).
		Build()
}
