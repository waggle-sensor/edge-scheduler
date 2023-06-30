package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type GPUAwareSchedulingPolicy struct {
}

func NewGPUAwareSchedulingPolicy() *GPUAwareSchedulingPolicy {
	return &GPUAwareSchedulingPolicy{}
}

// SelectBestPlugins returns the best plugin to run at the time
// For non-GPU-demand plugins, it returns all the plugins.
// For GPU-demand plugins it returns the oldest one if no GPU-demand plugins in the scheduled plugin list
func (rs *GPUAwareSchedulingPolicy) SelectBestPlugins(readyQueue *datatype.Queue, scheduledPlugins *datatype.Queue, availableResource datatype.Resource) (pluginsToRun []*datatype.PluginRuntime, err error) {
	GPUPluginExists := false
	// Flag if GPU-demand plugin already exists in scheduled plugin list
	scheduledPlugins.ResetIter()
	for scheduledPlugins.More() {
		pr := scheduledPlugins.Next()
		if pr.Plugin.PluginSpec.IsGPURequired() {
			GPUPluginExists = true
			logger.Debug.Printf("GPU-demand plugin %q exists in scheduled plugin list.", pr.Plugin.Name)
			break
		}
	}
	readyQueue.ResetIter()
	for readyQueue.More() {
		pr := readyQueue.Next()
		if pr.Plugin.PluginSpec.IsGPURequired() {
			if GPUPluginExists == false {
				pluginsToRun = append(pluginsToRun, pr)
				logger.Debug.Printf("GPU-demand plugin %q is added to scheduled plugin list.", pr.Plugin.Name)
				GPUPluginExists = true
			} else {
				logger.Debug.Printf("GPU-demand plugin %q needs to wait because other GPU-demand plugin is scheduled or being run.", pr.Plugin.Name)
			}
		} else {
			pluginsToRun = append(pluginsToRun, pr)
		}
	}
	return
}
