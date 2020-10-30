package datatype

import (
	"fmt"
)

// Plugin structs plugin metadata from ECR
type Plugin struct {
	Name         string    `yaml:"name,omitempty"`
	Version      string    `yaml:"version,omitempty"`
	Tags         []string  `yaml:"tags,omitempty"`
	Hardware     []string  `yaml:"hardware,omitempty"`
	Architecture []string  `yaml:"architecture,omitempty"`
	Profile      []Profile `yaml:"profile,omitempty"`
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
