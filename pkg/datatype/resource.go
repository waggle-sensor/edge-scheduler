package datatype

import (
	"regexp"
	"strconv"
	"strings"
)

// Resource structs resources used in scheduling
type Resource struct {
	CPU          string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory       string `json:"memory,omitempty" yaml:"memory,omitempty"`
	GPUMemory    string `json:"gpu_memory,omitempty" yaml:"gpuMemory,omitempty"`
	cpuInMilli   int    `json:"-" yaml:"-"`
	memInMega    int    `json:"-" yaml:"-"`
	gpuMemInMega int    `json:"-" yaml:"-"`
}

func (r *Resource) CanAccommodate(c *Resource) bool {
	r.convert()
	c.convert()
	if r.cpuInMilli >= c.cpuInMilli &&
		r.memInMega >= c.memInMega &&
		r.gpuMemInMega >= c.gpuMemInMega {
		return true
	} else {
		return false
	}
}

func (r *Resource) convert() {
	if strings.HasSuffix(r.CPU, "m") {
		if cpuInInt, err := strconv.Atoi(r.CPU[:len(r.CPU)-1]); err == nil {
			r.cpuInMilli = cpuInInt
		} else {
			r.cpuInMilli = -1
		}
	} else if cpuInInt, err := strconv.Atoi(r.CPU); err == nil {
		r.cpuInMilli = cpuInInt * 1000
	} else if cpuInF, err := strconv.ParseFloat(r.CPU, 32); err == nil {
		r.cpuInMilli = int(cpuInF * 1000.0)
	} else {
		r.cpuInMilli = -1
	}
	value, unit := splitValueAndUnit(r.Memory)
	switch unit {
	case "Ki":
		r.memInMega = int(value / 1024.)
	case "Mi":
		r.memInMega = value
	case "Gi":
		r.memInMega = value * 1024
	case "Ti":
		r.memInMega = value * 1024 * 1024
	}
	value, unit = splitValueAndUnit(r.GPUMemory)
	switch unit {
	case "Ki":
		r.gpuMemInMega = value * 1024
	case "Mi":
		r.gpuMemInMega = value
	case "Gi":
		r.gpuMemInMega = int(value / 1024.)
	case "Ti":
		r.gpuMemInMega = int(value / 1024. / 1024.)
	}
}

// splitValueAndUnit returns value and its unit. The unit is one of Ki, Mi, Gi, and Ti.
//
// If not unit is found, Mi is assumed.
func splitValueAndUnit(v string) (int, string) {
	r, _ := regexp.Compile("Ki|Mi|Gi|Ti")
	found := r.FindString(v)
	value, _ := strconv.Atoi(strings.ReplaceAll(v, found, ""))
	if found != "" {
		return value, found
	} else {
		return value, "Mi"
	}
}
