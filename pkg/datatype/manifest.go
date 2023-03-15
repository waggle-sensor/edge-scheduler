package datatype

import "strings"

// Node structs information about nodes
type NodeManifest struct {
	VSN      string            `json:"vsn" yaml:"vsn"`
	Name     string            `json:"name" yaml:"name"`
	Tags     []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Computes []ComputeManifest `json:"computes,omitempty" yaml:"computes,omitempty"`
	Sensors  []SensorManifest  `json:"sensors,omitempty" yaml:"sensors,omitempty"`
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

// GetPluginArchitectureSupportedComputes returns a compute device list that supports
// given plugin architecture
func (n *NodeManifest) GetPluginArchitectureSupportedComputes(pManifest *PluginManifest) (result bool, supportedComputes []ComputeManifest) {
	for _, c := range n.Computes {
		for _, pluginArch := range pManifest.GetArchitectures() {
			// NOTE: plugin manifest has linux/ prefix for architecture
			//       that node manifest does not have
			pluginArch = strings.Replace(pluginArch, "linux/", "", -1)
			if c.SupportsArchitecture(pluginArch) {
				supportedComputes = append(supportedComputes, c)
			}
		}
	}
	if len(supportedComputes) > 0 {
		result = true
	} else {
		result = false
	}
	return
}

// GetUnsupportedListOfPluginSensors returns a list of unsupported sensors by the node
func (n *NodeManifest) GetUnsupportedListOfPluginSensors(plugin *PluginManifest) (result bool, notSupported []string) {
	// TODO: plugin manifest does not yet have sensor list. Once it has a sensor list we will implement this function
	// for requiredHardware := range plugin.Hardware {
	// 	if _, exist := n.Hardware[requiredHardware]; !exist {
	// 		notSupported = append(notSupported, requiredHardware)
	// 	}
	// }
	// if len(notSupported) == 0 {
	// 	result = true
	// } else {
	// 	result = false
	// }
	return
}

// Device structs device specific meta information
type ComputeManifest struct {
	Name         string                  `json:"name" yaml:"name"`
	SerialNumber string                  `json:"serial_no" yaml:"serialNo"`
	Zone         string                  `json:"zone" yaml:"zone"`
	Hardware     ComputeHardwareManifest `json:"hardware" yaml:"hardware"`
}

func (c *ComputeManifest) GetArchitecture() string {
	return c.Hardware.GetArchitecture()
}

func (c *ComputeManifest) SupportsArchitecture(architecture string) bool {
	return c.Hardware.GetArchitecture() == architecture
}

// GetUnsupportedPluginProfiles returns available profiles that are
// supported by the node device
func (c *ComputeManifest) GetUnsupportedPluginProfiles(plugin *PluginManifest) (result bool, unsupportedProfiles []Profile) {
	// NOTE: if no profile is given, we assume the plugin can be run on the device
	// if len(plugin.Profile) < 1 {
	// 	result = true
	// 	return
	// }
	// result = false
	// for _, profile := range plugin.Profile {
	// 	if d.Resource.CanAccommodate(&profile.Require) {
	// 		result = true
	// 	} else {
	// 		unsupportedProfiles = append(unsupportedProfiles, profile)
	// 	}
	// }
	return
}

type ComputeHardwareManifest struct {
	Hardware     string   `json:"hardware" yaml:"hardware"`
	Model        string   `json:"hw_model" yaml:"hwModel"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
	CPU          string   `json:"cpu" yaml:"cpu"`
	CPURAM       string   `json:"cpu_ram" yaml:"cpuRam"`
	GPURAM       string   `json:"gpu_ram" yaml:"gpuRam"`
	SharedRam    bool     `json:"shared_ram" yaml:"sharedRam"`
}

// GetArchitecture returns architecture of the hardware.
// Capabilities should have one of arm64,amd64,armv7
func (c *ComputeHardwareManifest) GetArchitecture() string {
	archs := map[string]bool{
		"arm64": true,
		"amd64": true,
		"armv7": true,
	}
	for _, cap := range c.Capabilities {
		if _, found := archs[cap]; found {
			return cap
		}
	}
	return ""
}

type SensorManifest struct {
	Name   string   `json:"name" yaml:"name"`
	Scope  string   `json:"scope" yaml:"scope"`
	Labels []string `json:"labels" yaml:"labels"`
}

type SensorHardwareManifest struct {
	Hardware     string   `json:"hardware" yaml:"hardware"`
	Model        string   `json:"hw_model" yaml:"hwModel"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
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
	Architecture []string `json:"architectures" yaml:"architectures"`
	Branch       string   `json:"branch" yaml:"branch"`
}
