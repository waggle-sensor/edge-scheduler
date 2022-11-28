package datatype

import (
	"fmt"
	"path"
	"strings"
)

// Plugin structs plugin metadata from ECR
type Plugin struct {
	Name       string      `json:"name" yaml:"name"`
	PluginSpec *PluginSpec `json:"plugin_spec" yaml:"pluginSpec,omitempty"`
	DataShims  []*DataShim `json:"datathims,omitempty" yaml:"datashims,omitempty"`
	GoalID     string      `json:"goal_id,omitempty" yaml:"goalID,omitempty"`
}

func (p *Plugin) GetPluginImage() (string, error) {
	if p.PluginSpec == nil {
		return "", fmt.Errorf("Plugin does not have plugin spec")
	}
	return p.PluginSpec.Image, nil
}

type PluginSpec struct {
	Image       string            `json:"image" yaml:"image"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Privileged  bool              `json:"privileged,omitempty" yaml:"privileged,omitempty"`
	Node        string            `json:"node,omitempty" yaml:"node,omitempty"`
	Job         string            `json:"job,omitempty" yaml:"job,omitempty"`
	Selector    map[string]string `json:"selector,omitempty" yaml:"selector,omitempty"`
	Entrypoint  string            `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	DevelopMode bool              `json:"develop,omitempty" yaml:"develop,omitempty"`
}

func (ps *PluginSpec) GetImageTag() (string, error) {
	name := path.Base(ps.Image)
	parts := strings.Split(name, ":")

	switch len(parts) {
	case 1:
		return "latest", nil
	case 2:
		return parts[1], nil
	default:
		return "", fmt.Errorf("invalid image name")
	}
}

func (ps *PluginSpec) IsGPURequired() bool {
	if v, found := ps.Selector["resource.gpu"]; found {
		if v == "true" {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
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
