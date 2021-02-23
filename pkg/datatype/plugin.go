package datatype

import (
	"fmt"
)

// Plugin structs plugin metadata from ECR
type Plugin struct {
	Name         string       `yaml:"name,omitempty"`
	Version      string       `yaml:"version,omitempty"`
	Status       PluginStatus `yaml:"status,omitempty"`
	Tags         []string     `yaml:"tags,omitempty"`
	Hardware     []string     `yaml:"hardware,omitempty"`
	Architecture []string     `yaml:"architecture,omitempty"`
	DataShims    []*DataShim  `yaml:"datashims,omitempty"`
	Profile      []Profile    `yaml:"profile,omitempty"`
}

// PluginStatus structs status about a plugin that includes
// contexual, scheduling, and knob status
type PluginStatus struct {
	ContextStatus    ContextStatus    `yaml:"context,omitempty"`
	SchedulingStatus SchedulingStatus `yaml:"scheduling,omitempty"`
	KnobStatus       *Profile         `yaml:"knob,omitempty"`
}

// ContextStatus represents contextual status of a plugin
type ContextStatus string

const (
	// Runnable indicates a plugin is runnable wrt the current context
	Runnable ContextStatus = "Runnable"
	// Stoppable indicates a plugin is stoppable wrt the current context
	Stoppable = "Stoppable"
)

// SchedulingStatus represents scheduling status of a plugin
type SchedulingStatus string

const (
	// Running indicates a plugin is assigned resource and running
	Running SchedulingStatus = "Running"
	// Stopped indicates a plugin is stopped by scheduler
	Stopped = "Stopped"
)

// EventPluginContext structs a message about plugin context change
type EventPluginContext struct {
	GoalID     string        `json:"goal_id"`
	PluginName string        `json:"plugin_name"`
	Status     ContextStatus `json:"status"`
}

type PluginCredential struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// type Plugin struct {
// 	Name      string   `yaml:"name"`
// 	Image     string   `yaml:"image"`
// 	Ports     []Port   `yaml:"ports"`
// 	Args      []string `yaml:"args"`
// 	Resources struct {
// 		Requests []Resource `yaml:"requests"`
// 		Limits   []Resource `yaml:"limits"`
// 	}
// 	Env     map[string]string `yaml:"env"`
// 	Configs map[string]string `yaml:"configs"`
// }

// UpdatePluginContext updates contextual status of the plugin
func (p *Plugin) UpdatePluginContext(status ContextStatus) error {
	p.Status.ContextStatus = status
	return nil
}

// RemoveProfile removes an existing Profile by name
func (p *Plugin) RemoveProfile(profileToBeDeleted Profile) error {
	for index, profile := range p.Profile {
		if profile.Name == profileToBeDeleted.Name {
			p.Profile = append(p.Profile[:index], p.Profile[index+1:]...)
			return nil
		}
	}
	return fmt.Errorf("Profile %s not found", profileToBeDeleted.Name)
}

// Profile structs a name with knobs
type Profile struct {
	Name    string            `yaml:"name,omitempty"`
	Knobs   map[string]string `yaml:"knobs,omitempty"`
	Require Resource          `yaml:"require,omitempty"`
}
