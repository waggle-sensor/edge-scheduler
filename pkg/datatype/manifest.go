package datatype

// Node structs information about nodes
type NodeManifest struct {
	Name     string                 `json:"name" yaml:"name"`
	Tags     []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Devices  []Device               `json:"devices,omitempty" yaml:"devices,omitempty"`
	Hardware map[string]interface{} `json:"hardware,omitempty" yaml:"hardware,omitempty"`
	Ontology map[string]interface{} `json:"ontology,omitempty" yaml:"ontology,omitempty"`
}

func (n *NodeManifest) MatchTags(tags []string, matchAll bool) bool {
	tagCount := 0
	for _, tag := range tags {
		for _, nodeTag := range n.Tags {
			if nodeTag == tag {
				tagCount += 1
				break
			}
		}
	}
	if matchAll {
		if tagCount == len(tags) {
			return true
		} else {
			return false
		}
	} else {
		if tagCount > 0 {
			return true
		} else {
			return false
		}
	}
}

// GetPluginArchitectureSupportedDevices returns a device list that supports
// plugin architecture
func (n *NodeManifest) GetPluginArchitectureSupportedDevices(pManifest *PluginManifest) (result bool, supportedDevices []Device) {
	for _, nodeDevice := range n.Devices {
		for _, pluginArch := range pManifest.GetArchitectures() {
			if pluginArch == nodeDevice.Architecture {
				supportedDevices = append(supportedDevices, nodeDevice)
			}
		}
	}
	if len(supportedDevices) > 0 {
		result = true
	} else {
		result = false
	}
	return
}

// GetPluginHardwareUnsupportedList returns a list of hardware that are not
// supported by the node
func (n *NodeManifest) GetPluginHardwareUnsupportedList(plugin *PluginManifest) (result bool, notSupported []string) {
	for requiredHardware := range plugin.Hardware {
		if _, exist := n.Hardware[requiredHardware]; !exist {
			notSupported = append(notSupported, requiredHardware)
		}
	}
	if len(notSupported) == 0 {
		result = true
	} else {
		result = false
	}
	return
}

// Device structs device specific meta information
type Device struct {
	Name         string   `yaml:"name,omitempty"`
	Architecture string   `yaml:"architecture,omitempty"`
	Resource     Resource `yaml:"resource,omitempty"`
}

// GetUnsupportedPluginProfiles returns available profiles that are
// supported by the node device
func (d *Device) GetUnsupportedPluginProfiles(plugin *PluginManifest) (result bool, unsupportedProfiles []Profile) {
	// NOTE: if no profile is given, we assume the plugin can be run on the device
	if len(plugin.Profile) < 1 {
		result = true
		return
	}
	result = false
	for _, profile := range plugin.Profile {
		if d.Resource.CanAccommodate(&profile.Require) {
			result = true
		} else {
			unsupportedProfiles = append(unsupportedProfiles, profile)
		}
	}
	return
}

type PluginManifest struct {
	Name     string               `json:"name" yaml:"name"`
	ID       string               `json:"id" yaml:"id"`
	Tags     map[string]bool      `json:"tags,omitempty" yaml:"tags,omitempty"`
	Source   PluginManifestSource `json:"source" yaml:"source"`
	Hardware map[string]bool      `json:"required_hardware,omitempty" yaml:"requiredHardware,omitempty"`
	Profile  []Profile            `json:"profiles,omitempty" yaml:"profiles,omitempty"`
}

func (p *PluginManifest) GetArchitectures() []string {
	return p.Source.Architecture
}

type PluginManifestSource struct {
	Architecture []string `json:"architecture" yaml:"architecture"`
	Branch       string   `json:"branch" yaml:"branch"`
}
