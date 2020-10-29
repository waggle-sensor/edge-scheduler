package node

// Node structs information about nodes
type Node struct {
	Name     string                 `yaml:"name,omitempty"`
	Tags     []string               `yaml:"tags,omitempty"`
	Devices  []Device               `yaml:"devices,omitempty"`
	Hardware []string               `yaml:"hardware,omitempty"`
	Ontology map[string]interface{} `yaml:"ontology,omitempty"`
}

// GetPluginArchitectureSupportedDevices returns a device list that supports
// plugin architecture
func (n *Node) GetPluginArchitectureSupportedDevices(plugin Plugin) (result bool, supportedDevices []Device) {
	for _, nodeDevice := range n.Devices {
		for _, pluginArch := range plugin.Architecture {
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
func (n *Node) GetPluginHardwareUnsupportedList(plugin Plugin) (result bool, notSupported []string) {
	for _, requiredHardware := range plugin.Hardware {
		supported := false
		for _, hardware := range n.Hardware {
			if requiredHardware == hardware {
				supported = true
				break
			}
		}
		if !supported {
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
func (d *Device) GetUnsupportedPluginProfiles(plugin Plugin) (result bool, unsupportedProfiles []Profile) {
	result = false
	for _, profile := range plugin.Profile {
		if profile.Require.CPU > d.Resource.CPU ||
			profile.Require.Memory > d.Resource.Memory ||
			profile.Require.GPUMemory > d.Resource.GPUMemory {
			unsupportedProfiles = append(unsupportedProfiles, profile)
		} else {
			result = true
		}
	}
	return
}
