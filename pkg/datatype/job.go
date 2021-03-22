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
	ScienceRules       []string `yaml:"sciencerules,omitempty"`
}

// TODO: This prohibits running same plugins. Is this appropriate?
// AddPlugin adds given plugin to the job
func (j *Job) AddPlugin(plugin Plugin) {
	for _, p := range j.Plugins {
		if p.Name == plugin.Name && p.Version == plugin.Version {
			return
		}
	}
	j.Plugins = append(j.Plugins, plugin)
}

// AddNode adds given node to the job
func (j *Job) AddNode(node Node) {
	for _, n := range j.Nodes {
		if n.Name == node.Name {
			return
		}
	}
	j.Nodes = append(j.Nodes, node)
}
