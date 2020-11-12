package datatype

// ScienceGoal structs local goals and success criteria
type ScienceGoal struct {
	ID         string    `yaml:"id"`
	Name       string    `yaml:"name,omitempty"`
	SubGoals   []SubGoal `yaml:"subgoals,omitempty"`
	Conditions []string  `yaml:"conditions,omitempty"`
}

// GetMySubGoal returns the subgoal assigned to node
func (sg *ScienceGoal) GetMySubGoal(nodeName string) *SubGoal {
	for _, subGoal := range sg.SubGoals {
		if subGoal.Node.Name == nodeName {
			return &subGoal
		}
	}
	return nil
}

// SubGoal structs node-specific goal along with conditions and rules
type SubGoal struct {
	Node       Node     `yaml:"node,omitempty"`
	Plugins    []Plugin `yaml:"plugins,omitempty"`
	Rules      []string `yaml:"rules,omitempty"`
	Statements []string `yaml:"statements,omitempty"`
}

// type Goal struct {
// 	APIVersion string `yaml:"apiVersion"`
// 	Header     struct {
// 		GoalId      string   `yaml:"goalId"`
// 		GoalName    string   `yaml:"goalName"`
// 		Priority    int      `yaml:"priority"`
// 		TargetNodes []string `yaml:"targetNodes"`
// 		UserId      string   `yaml:"userId"`
// 	}
// 	Body struct {
// 		AppConfig    []PluginConfig `yaml:"appConfig"`
// 		Rules        []string       `yaml:"rules"`
// 		SensorConfig struct {
// 			Plugins []Plugin `yaml:"plugins"`
// 		}
// 	}
// }
