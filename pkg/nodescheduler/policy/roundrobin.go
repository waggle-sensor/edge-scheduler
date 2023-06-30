package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type RoundRobinSchedulingPolicy struct {
}

func NewRoundRobinSchedulingPolicy() *RoundRobinSchedulingPolicy {
	return &RoundRobinSchedulingPolicy{}
}

// SelectBestPlugins returns the best plugin to run at the time
// It returns the oldest plugin amongst "ready" plugins
func (rs *RoundRobinSchedulingPolicy) SelectBestPlugins(readyQueue *datatype.Queue, scheduledPlugins *datatype.Queue, availableResource datatype.Resource) (pluginsToRun []*datatype.PluginRuntime, err error) {
	if scheduledPlugins.Length() > 0 {
		return
	}
	// Pick up the oldest Ready plugin
	p := readyQueue.PopFirst()
	if p != nil {
		pluginsToRun = append(pluginsToRun, p)
	}
	return
}
