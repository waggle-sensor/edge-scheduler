package datatype

// Node structs information about nodes
type NodeManifest struct {
	Name     string                 `json:"name" yaml:"name"`
	Tags     map[string]bool        `json:"tags,omitempty" yaml:"tags,omitempty"`
	Devices  []Device               `json:"devices,omitempty" yaml:"devices,omitempty"`
	Hardware map[string]interface{} `json:"hardware,omitempty" yaml:"hardware,omitempty"`
	Ontology map[string]interface{} `json:"ontology,omitempty" yaml:"ontology,omitempty"`
}

// GetPluginArchitectureSupportedDevices returns a device list that supports
// plugin architecture
func (n *NodeManifest) GetPluginArchitectureSupportedDevices(plugin *PluginManifest) (result bool, supportedDevices []Device) {
	for _, nodeDevice := range n.Devices {
		for pluginArch := range plugin.Architecture {
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
	Name         string                 `json:"name" yaml:"name"`
	Image        string                 `json:"image" yaml:"image"`
	Tags         map[string]bool        `json:"tags,omitempty" yaml:"tags,omitempty"`
	Hardware     map[string]bool        `json:"required_hardware,omitempty" yaml:"requiredHardware,omitempty"`
	Architecture map[string]interface{} `json:"arch" yaml:"arch"`
	Profile      []Profile              `json:"profiles,omitempty" yaml:"profiles,omitempty"`
}
