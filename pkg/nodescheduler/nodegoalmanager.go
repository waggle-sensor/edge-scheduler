package nodescheduler

import (
	"fmt"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

// GoalManager structs a goal manager for nodescheduler
type NodeGoalManager struct {
	ScienceGoals map[string]datatype.ScienceGoal
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
