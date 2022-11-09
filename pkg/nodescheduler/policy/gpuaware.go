package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type GPUAwareSchedulingPolicy struct {
}

func NewGPUAwareSchedulingPolicy() *GPUAwareSchedulingPolicy {
	return &GPUAwareSchedulingPolicy{}
}

// SelectBestPlugins returns the best plugin to run at the time
// For non-GPU-demand plugins, it returns all the plugins.
// For GPU-demand plugins it returns the oldest one if no GPU-demand plugins in the scheduled plugin list
func (rs *GPUAwareSchedulingPolicy) SelectBestPlugins(readyQueue *datatype.Queue, scheduledPlugins *datatype.Queue, availableResource datatype.Resource) (pluginsToRun []*datatype.Plugin, err error) {
	GPUPluginExists := false
	// Flag if GPU-demand plugin already exists in scheduled plugin list
	scheduledPlugins.ResetIter()
	for scheduledPlugins.More() {
		p := scheduledPlugins.Next()
		if p.PluginSpec.IsGPURequired() {
			GPUPluginExists = true
			break
		}
	}
	readyQueue.ResetIter()
	for readyQueue.More() {
		p := readyQueue.Next()
		if p.PluginSpec.IsGPURequired() {
			if GPUPluginExists == false {
				readyQueue.Pop(p)
				pluginsToRun = append(pluginsToRun, p)
				GPUPluginExists = true
			}
		} else {
			readyQueue.Pop(p)
			pluginsToRun = append(pluginsToRun, p)
		}
	}
	return
}
