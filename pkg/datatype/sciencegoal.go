package datatype

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	uuid "github.com/nu7hatch/gouuid"
)

type ScienceGoalBuilder struct {
	sg ScienceGoal
}

func NewScienceGoalBuilder(goalName string, jobID string) *ScienceGoalBuilder {
	id, _ := uuid.NewV4()
	return &ScienceGoalBuilder{
		sg: ScienceGoal{
			ID:    id.String(),
			JobID: jobID,
			Name:  goalName,
		},
	}
}

func (sgb *ScienceGoalBuilder) AddSubGoal(nodeID string, plugins []*Plugin, scienceRules []string) *ScienceGoalBuilder {
	subGoal := &SubGoal{
		Name:         nodeID,
		Plugins:      plugins,
		ScienceRules: scienceRules,
	}
	subGoal.ApplyGoalIDToPlugins(sgb.sg.ID)
	err := subGoal.AddChecksum()
	if err != nil {
		return nil
	}
	sgb.sg.SubGoals = append(sgb.sg.SubGoals, subGoal)
	return sgb
}

func (sgb *ScienceGoalBuilder) Build() *ScienceGoal {
	return &sgb.sg
}

// ScienceGoal structs local goals and success criteria
type ScienceGoal struct {
	ID         string     `json:"id" yaml:"id"`
	JobID      string     `json:"job_id" yaml:"jobID"`
	Name       string     `json:"name,omitempty" yaml:"name,omitempty"`
	SubGoals   []*SubGoal `json:"sub_goals,omitempty" yaml:"subgoals,omitempty"`
	Conditions []string   `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// GetMySubGoal returns the subgoal assigned to node
func (g *ScienceGoal) GetMySubGoal(nodeName string) *SubGoal {
	for _, subGoal := range g.SubGoals {
		if strings.ToLower(subGoal.Name) == strings.ToLower(nodeName) {
			return subGoal
		}
	}
	return nil
}

// ShowMyScienceGoal returns the science goal with node's sub goal.
// It simply removes all other nodes' sub goal. Note that this creates a new object (deep copy).
func (g *ScienceGoal) ShowMyScienceGoal(nodeName string) *ScienceGoal {
	mySubgoal := *g.GetMySubGoal(nodeName)
	return &ScienceGoal{
		ID:         g.ID,
		JobID:      g.JobID,
		Name:       g.Name,
		SubGoals:   []*SubGoal{&mySubgoal},
		Conditions: g.Conditions,
	}
}

// GetSubjectNodes returns a list of nodes subject to run this science goal
func (g *ScienceGoal) GetSubjectNodes() (nodes []string) {
	for _, subGoal := range g.SubGoals {
		nodes = append(nodes, subGoal.Name)
	}
	return
}

// SubGoal structs node-specific goal along with conditions and rules
type SubGoal struct {
	Name         string    `json:"name" yaml:"name"`
	Plugins      []*Plugin `json:"plugins" yaml:"plugins"`
	ScienceRules []string  `json:"science_rules" yaml:"scienceRules"`
	checksum     string    `json:"-" yaml:"-"`
}

// func NewSubGoal(goalID string, nodeID string, plugins []*PluginSpec) *SubGoal {
// 	var subGoal SubGoal
// 	subGoal.Node = &Node{
// 		Name: nodeID,
// 	}
// 	for _, pluginSpec := range plugins {
// 		subGoal.Plugins = append(subGoal.Plugins, &Plugin{
// 			Name:       pluginSpec.Name,
// 			PluginSpec: pluginSpec,
// 			Status: PluginStatus{
// 				SchedulingStatus: Waiting,
// 			},
// 		})
// 	}
// 	specjson, err := json.Marshal(subGoal)
// 	if err != nil {
// 		return nil
// 	}
// 	sum := sha256.Sum256(specjson)
// 	// instance := hex.EncodeToString(sum[:])[:8]
// 	subGoal.checksum = hex.EncodeToString(sum[:])
// 	return &subGoal
// }

func (sg *SubGoal) AddChecksum() error {
	specjson, err := json.Marshal(sg)
	if err != nil {
		// We cannot proceed anymore
		return err
	}
	sum := sha256.Sum256(specjson)
	// instance := hex.EncodeToString(sum[:])[:8]
	sg.checksum = hex.EncodeToString(sum[:])
	return nil
}

func (sg *SubGoal) CompareChecksum(otherSubGoal *SubGoal) bool {
	if sg.checksum == otherSubGoal.checksum {
		return true
	} else {
		return false
	}
}

func (sg *SubGoal) ApplyGoalIDToPlugins(goalID string) {
	for _, plugin := range sg.Plugins {
		plugin.GoalID = goalID
	}
}

// UpdatePluginContext updates plugin's context event within the subgoal
// It returns an error if it fails to update context status of the plugin
// func (sg *SubGoal) UpdatePluginContext(contextEvent EventPluginContext) error {
// 	for _, plugin := range sg.Plugins {
// 		if plugin.Name == contextEvent.PluginName {
// 			return plugin.UpdatePluginContext(contextEvent.Status)
// 		}
// 	}
// 	return fmt.Errorf("failed to update context (%s) of plugin %s", contextEvent.Status, contextEvent.PluginName)
// }

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

func (sg *SubGoal) AddPlugin(plugin *Plugin) {
	sg.Plugins = append(sg.Plugins, plugin)
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
	Plugins      []*Plugin
	ScienceRules []string `yaml:"science_rules"`
}

func (j *JobTemplate) ConvertJobTemplateToScienceGoal(nodeID string) *ScienceGoal {
	return NewScienceGoalBuilder(j.Name, "-1").
		AddSubGoal(nodeID, j.Plugins, j.ScienceRules).
		Build()
}
