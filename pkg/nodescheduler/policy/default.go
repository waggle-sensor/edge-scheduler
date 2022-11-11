package policy

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type SchedulingPolicy interface {
	SelectBestPlugins(*datatype.Queue, *datatype.Queue, datatype.Resource) ([]*datatype.Plugin, error)
}

func GetSchedulingPolicyByName(policyName string) SchedulingPolicy {
	switch policyName {
	case "default":
		logger.Info.Println("Default policy is selected")
		return NewSimpleSchedulingPolicy()
	case "roundrobin":
		logger.Info.Println("Round-robin policy is selected")
		return NewRoundRobinSchedulingPolicy()
	case "gpuaware":
		logger.Info.Println("GPU-aware policy is selected")
		return NewGPUAwareSchedulingPolicy()
	default:
		logger.Error.Printf("Given policy name %q does not exist. Default policy is selected", policyName)
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
