package datatype

// ScienceGoal structs local goals and success criteria
type ScienceGoal struct {
	ID         string    `yaml:"id"`
	Name       string    `yaml:"name,omitempty"`
	SubGoals   []SubGoal `yaml:"subgoals,omitempty"`
	Conditions []string  `yaml:"conditions,omitempty"`
}

// SubGoal structs node-specific goal along with conditions and rules
type SubGoal struct {
	Node       Node     `yaml:"node,omitempty"`
	Plugins    []Plugin `yaml:"plugins,omitempty"`
	Conditions []string `yaml:"conditions,omitempty"`
	Rules      []string `yaml:"rules,omitempty"`
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
