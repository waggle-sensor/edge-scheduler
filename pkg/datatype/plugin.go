package datatype

import (
	"path"
	"strings"
	"time"
)

// Plugin structs plugin metadata from ECR
type Plugin struct {
	Name       string       `json:"name" yaml:"name"`
	PluginSpec *PluginSpec  `json:"plugin_spec" yaml:"pluginSpec,omitempty"`
	Status     PluginStatus `json:"status,omitempty" yaml:"status,omitempty"`
	DataShims  []*DataShim  `yaml:"datashims,omitempty"`
	GoalID     string       `json:"-" yaml:"-"`
}

type PluginSpec struct {
	Image       string            `json:"image" yaml:"image"`
	Args        []string          `json:"args" yaml:"args"`
	Privileged  bool              `json:"privileged,omitempty" yaml:"privileged,omitempty"`
	Node        string            `json:"node" yaml:"node"`
	Job         string            `json:"job,omitempty" yaml:"job,omitempty"`
	Selector    map[string]string `json:"selector" yaml:"selector"`
	Entrypoint  string            `json:"entrypoint" yaml:"entrypoint"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	DevelopMode bool              `json:"develop,omitempty" yaml:"develop,omitempty"`
}

func (ps *PluginSpec) GetImageVersion() string {
	// split name:version from image string
	parts := strings.Split(path.Base(ps.Image), ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// PluginStatus structs status about a plugin that includes
// contexual, scheduling, and knob status
type PluginStatus struct {
	ContextStatus    ContextStatus    `json:"context,omitempty" yaml:"context,omitempty"`
	SchedulingStatus SchedulingStatus `json:"scheduling,omitempty" yaml:"scheduling,omitempty"`
	KnobStatus       *Profile         `json:"knob,omitempty" yaml:"knob,omitempty"`
	Since            time.Time        `json:"since,omitempty" yaml:"since,omitempty"`
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
	// Waiting indicates a plugin is not activated and in waiting
	Waiting SchedulingStatus = "Waiting"
	// Ready indicates a plugin is ready to be scheduled
	Ready SchedulingStatus = "Ready"
	// Running indicates a plugin is assigned resource and running
	Running SchedulingStatus = "Running"
	// Stopped indicates a plugin is stopped by scheduler
	Stopped SchedulingStatus = "Stopped"
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
func (p *Plugin) UpdatePluginSchedulingStatus(status SchedulingStatus) error {
	p.Status.SchedulingStatus = status
	p.Status.Since = time.Now()
	return nil
}

// GetMetaInformation returns a dictionary of plugin information.
// func (p *Plugin) GetMetaInformation() map[string]string {
// 	meta := make(map[string]string)
// 	meta["Name"] = p.Name
// 	meta["Image"] = p.PluginSpec.Image
// 	meta["Status"] = string(p.Status.SchedulingStatus)
// 	meta["Task"] = p.PluginSpec.Job
// 	meta["Args"] = strings.Join(p.PluginSpec.Args, " ")
// 	meta["Selector"] = fmt.Sprint(p.PluginSpec.Selector)
// 	return meta
// }

// UpdatePluginContext updates contextual status of the plugin
func (p *Plugin) UpdatePluginContext(status ContextStatus) error {
	p.Status.ContextStatus = status
	return nil
}

// RemoveProfile removes an existing Profile by name
// func (p *Plugin) RemoveProfile(profileToBeDeleted Profile) error {
// 	for index, profile := range p.Profile {
// 		if profile.Name == profileToBeDeleted.Name {
// 			p.Profile = append(p.Profile[:index], p.Profile[index+1:]...)
// 			return nil
// 		}
// 	}
// 	return fmt.Errorf("Profile %s not found", profileToBeDeleted.Name)
// }

// Profile structs a name with knobs
type Profile struct {
	Name    string            `yaml:"name,omitempty"`
	Knobs   map[string]string `yaml:"knobs,omitempty"`
	Require Resource          `yaml:"require,omitempty"`
}
