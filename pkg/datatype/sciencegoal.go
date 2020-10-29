package sciencegoal

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
