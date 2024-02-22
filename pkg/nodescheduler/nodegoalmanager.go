package nodescheduler

import (
	"fmt"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type PluginIndex struct {
	jobID  string
	goalID string
	name   string
}

// GoalManager structs a goal manager for nodescheduler
type NodeGoalManager struct {
	ScienceGoals  map[string]datatype.ScienceGoal
	LoadedPlugins map[PluginIndex]*datatype.PluginRuntime
}

// GetScienceGoalByID returns the goal of given goal name
func (ngm *NodeGoalManager) GetScienceGoalByID(goalID string) (*datatype.ScienceGoal, error) {
	if goal, exist := ngm.ScienceGoals[goalID]; exist {
		return &goal, nil
	}

	return nil, fmt.Errorf("The goal ID %s does not exist", goalID)
}

// GetScienceGoalByJobID returns the goal serving given job ID
func (ngm *NodeGoalManager) GetScienceGoalByJobID(jobID string) (*datatype.ScienceGoal, error) {
	for _, goal := range ngm.ScienceGoals {
		if goal.JobID == jobID {
			return &goal, nil
		}
	}
	return nil, fmt.Errorf("There is no goal serving the job %q", jobID)
}

func (ngm *NodeGoalManager) GetScienceGoalByName(goalName string) (*datatype.ScienceGoal, error) {
	for _, goal := range ngm.ScienceGoals {
		if goal.Name == goalName {
			return &goal, nil
		}
	}
	return nil, fmt.Errorf("The goal Name %s does not exist", goalName)
}

// DropGoal drops given goal from the list
func (ngm *NodeGoalManager) DropGoalByName(goalName string) error {
	for _, goal := range ngm.ScienceGoals {
		if goal.Name == goalName {
			delete(ngm.ScienceGoals, goal.ID)
			return nil
		}
	}
	return fmt.Errorf("The goal %s does not exist", goalName)
}

func (ngm *NodeGoalManager) DropGoal(goalID string) error {
	if _, exist := ngm.ScienceGoals[goalID]; exist {
		delete(ngm.ScienceGoals, goalID)
		return nil
	} else {
		return fmt.Errorf("Failed to find goal by ID %s", goalID)
	}
}

func (ngm *NodeGoalManager) AddGoal(goal *datatype.ScienceGoal) {
	ngm.ScienceGoals[goal.ID] = *goal
}

func (ngm *NodeGoalManager) AddPluginRuntime(p *datatype.PluginRuntime) {
	index := PluginIndex{
		jobID:  p.Plugin.JobID,
		goalID: p.Plugin.GoalID,
		name:   p.Plugin.Name,
	}
	ngm.LoadedPlugins[index] = p
}

func (ngm *NodeGoalManager) DropPluginRuntime(index PluginIndex) {
	delete(ngm.LoadedPlugins, index)
}

func (ngm *NodeGoalManager) GetPluginRuntime(index PluginIndex) *datatype.PluginRuntime {
	if p, found := ngm.LoadedPlugins[index]; found {
		return p
	} else {
		return nil
	}
}

// GetPluginRuntimeByNameAndJobID returns PluginRuntime of the plugin if exists.
// It attempts to get GoalID from registered jobs. If goal not found, returns nil.
func (ngm *NodeGoalManager) GetPluginRuntimeByNameAndJobID(name string, jobID string) *datatype.PluginRuntime {
	g, err := ngm.GetScienceGoalByJobID(jobID)
	if err != nil {
		logger.Error.Printf("failed to get goal by job ID %q: %s", jobID, err.Error())
		return nil
	}
	return ngm.GetPluginRuntime(PluginIndex{
		name:   name,
		goalID: g.ID,
		jobID:  jobID,
	})
}

func (ngm *NodeGoalManager) GetPluginRuntimeByPodUID(uid string) *datatype.PluginRuntime {
	for _, pr := range ngm.LoadedPlugins {
		if pr.PodUID == uid {
			return pr
		}
	}
	return nil
}

func (ngm *NodeGoalManager) GetQueuedPluginRuntime() (r []*datatype.PluginRuntime) {
	for _, pr := range ngm.LoadedPlugins {
		if pr.Status.State == datatype.Queued {
			r = append(r, pr)
		}
	}
	return
}
