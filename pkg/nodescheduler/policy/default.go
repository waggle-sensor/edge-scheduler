package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type SchedulingPolicy interface {
	SelectBestPlugins(*datatype.Queue, *datatype.Queue, datatype.Resource) ([]*datatype.Plugin, error)
}

func GetSchedulingPolicyByName(policyName string) SchedulingPolicy {
	switch policyName {
	case "default":
		return NewSimpleSchedulingPolicy()
	case "roundrobin":
		return NewRoundRobinSchedulingPolicy()
	case "gpuaware":
		return NewGPUAwareSchedulingPolicy()
	default:
		return NewSimpleSchedulingPolicy()
	}
}

type SimpleSchedulingPolicy struct {
}

func NewSimpleSchedulingPolicy() *SimpleSchedulingPolicy {
	return &SimpleSchedulingPolicy{}
}

// SelectBestPlugins returns the best plugin to run at the time
// For SimpleSchedulingPolicy, it returns all "ready" plugins
func (ss *SimpleSchedulingPolicy) SelectBestPlugins(readyQueue *datatype.Queue, scheduledPlugins *datatype.Queue, availableResource datatype.Resource) (pluginsToRun []*datatype.Plugin, err error) {
	readyQueue.ResetIter()
	for readyQueue.More() {
		pluginsToRun = append(pluginsToRun, readyQueue.Next())
	}
	return pluginsToRun, nil
}
