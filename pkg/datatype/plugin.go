package datatype

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"

	"github.com/looplab/fsm"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// Plugin structs plugin metadata from ECR
type Plugin struct {
	Name       string      `json:"name" yaml:"name"`
	PluginSpec *PluginSpec `json:"plugin_spec" yaml:"pluginSpec,omitempty"`
	GoalID     string      `json:"goal_id,omitempty" yaml:"goalID,omitempty"`
	JobID      string      `json:"job_id,omitempty" yaml:"jobID,omitempty"`
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
	Resource    map[string]string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Volume      map[string]string `json:"volume,omitempty" yaml:"volume,omitempty`
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
	Stoppable ContextStatus = "Stoppable"
)

// EventPluginContext structs a message about plugin context change
type EventPluginContext struct {
	GoalID     string        `json:"goal_id"`
	PluginName string        `json:"plugin_name"`
	Status     ContextStatus `json:"status"`
}

// PluginState is a label of Plugin state
type PluginState string

const (
	// Inactive indicates that plugin is not considered for scheduling
	// Science rules when valid can make a transition for plugin
	Inactive PluginState = "inactive"
	// Queued indicates that plugin is waiting for resource to be executed
	Queued PluginState = "queued"
	// Scheduled indicates that plugin is assigned resource and is created
	// in the system
	Scheduled PluginState = "scheduled"
	// Initializing indicates that plugin is being initialized
	Initializing PluginState = "initializing"
	// Running indicates that plugin's main container starts to run
	Running PluginState = "running"
	// Completed indicates that plugin has completed its run by examining
	// return code from plugin's main container, only for 0 as a return code
	Completed PluginState = "completed"
	// Failed indicates that plugin has failed to reach to Completed state.
	// Many reasons can transition to this state including,
	//
	// - Scheduled fails to create plugin in the system
	//
	// - Initializing fails to initialize plugin containers
	//
	// - Running fails to run plugin's program or the program exited with non-zero return code
	Failed PluginState = "failed"
)

type PluginStatus struct {
	State       PluginState `json:"state" yaml:"state"`
	LastState   PluginState `json:"last_state" yaml:"lastState"`
	LastUpdated time.Time   `json:"last_updated" yaml:"lastUpdated"`
}

type PluginCredential struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type PluginRuntime struct {
	Plugin                 Plugin
	Duration               int
	EnablePluginController bool
	Resource               Resource
	PodUID                 string
	Status                 *fsm.FSM
	PodInstance            string
}

func NewPluginRuntime(p Plugin) *PluginRuntime {
	pr := &PluginRuntime{
		Plugin: p,
		// Creating a finite state machine for PluginRuntme
		Status: fsm.NewFSM(
			string(Inactive),
			fsm.Events{
				{
					Name: string(Queued),
					Src:  []string{string(Inactive)},
					Dst:  string(Queued),
				},
				{
					Name: string(Scheduled),
					Src:  []string{string(Queued)},
					Dst:  string(Scheduled),
				},
				{
					Name: string(Initializing),
					Src:  []string{string(Scheduled)},
					Dst:  string(Initializing),
				},
				{
					Name: string(Running),
					Src:  []string{string(Initializing)},
					Dst:  string(Running),
				},
				{
					Name: string(Completed),
					Src:  []string{string(Running)},
					Dst:  string(Completed),
				},
				{
					Name: string(Failed),
					Src:  []string{string(Initializing), string(Running)},
					Dst:  string(Failed),
				},
				{
					Name: string(Inactive),
					Src:  []string{string(Queued), string(Scheduled), string(Initializing), string(Running), string(Completed), string(Failed)},
					Dst:  string(Inactive),
				},
			},
			fsm.Callbacks{},
		),
	}
	return pr
}

func NewPluginRuntimeWithScienceRule(p Plugin, runtimeArgs ScienceRule) *PluginRuntime {
	pr := NewPluginRuntime(p)
	pr.UpdateWithScienceRule(runtimeArgs)
	return pr
}

func (pr *PluginRuntime) SetPluginController(flag bool) {
	pr.EnablePluginController = flag
}

func (pr *PluginRuntime) UpdateWithScienceRule(runtimeArgs ScienceRule) {
	// TODO: any runtime parameters of the plugin should be parsed and added to the runtime
	// if v, found := runtimeArgs.ActionParameters["duration"]; found {
	// 	pr.Duration
	// }
}

// GeneratePodInstance generates a PodInstance of the PluginRuntime.
// The format consists of <<plugin name>>-<<6 random characters>>.
func (pr *PluginRuntime) GeneratePodInstance() {
	pr.PodInstance = pr.Plugin.Name + "-" + generateRandomString(6)
}

// Equal checks if given PluginRuntime object is the same in terms of its content.
// It checks Plugin's name, JobID, and GoalID. If matched, returns true, else false.
func (pr *PluginRuntime) Equal(_pr *PluginRuntime) bool {
	return pr.Plugin.Name == _pr.Plugin.Name &&
		pr.Plugin.JobID == _pr.Plugin.JobID &&
		pr.Plugin.GoalID == _pr.Plugin.GoalID
}

// func (pr *PluginRuntime) UpdateState(s PluginState) {
// 	lastState := pr.Status.Current()
// 	pr.Status.State = s
// 	pr.Status.LastUpdated = time.Now()
// 	pr.Status.LastState = lastState
// }

func (pr *PluginRuntime) SetPodUID(UID string) {
	pr.PodUID = UID
}

func (pr *PluginRuntime) Inactive() error {
	return pr.Status.Event(context.Background(), string(Inactive))
}

func (pr *PluginRuntime) Queued() error {
	return pr.Status.Event(context.Background(), string(Queued))
}

func (pr *PluginRuntime) Scheduled() error {
	return pr.Status.Event(context.Background(), string(Scheduled))
}

func (pr *PluginRuntime) Initializing() error {
	return pr.Status.Event(context.Background(), string(Initializing))
}

func (pr *PluginRuntime) Running() error {
	return pr.Status.Event(context.Background(), string(Running))
}

func (pr *PluginRuntime) Completed() error {
	return pr.Status.Event(context.Background(), string(Completed))
}

func (pr *PluginRuntime) Failed() error {
	return pr.Status.Event(context.Background(), string(Failed))
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

func generateRandomString(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
