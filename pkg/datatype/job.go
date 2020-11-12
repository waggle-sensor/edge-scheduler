package datatype

// Job structs user request for jobs
type Job struct {
	ID                 string   `yaml:"id"`
	Name               string   `yaml:"name,omitempty"`
	PluginTags         []string `yaml:"plugintags,omitempty"`
	Plugins            []Plugin `yaml:"plugins,omitempty"`
	NodeTags           []string `yaml:"nodetags,omitempty"`
	Nodes              []Node   `yaml:"nodes,omitempty"`
	MinimumPerformance []string `yaml:"minimumperformance,omitempty"`
	Rules              []string `yaml:"rules,omitempty"`
	Statements         []string `yaml:"statements,omitempty"`
}
