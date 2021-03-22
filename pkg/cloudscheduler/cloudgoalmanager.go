package cloudscheduler

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/datatype"
)

// CloudGoalManager structs a goal manager for cloudscheduler
type CloudGoalManager struct {
	scienceGoals map[string]*datatype.ScienceGoal
}

// NewCloudGoalManager returns an instance of cloud goal manager
func NewCloudGoalManager() (*CloudGoalManager, error) {
	return &CloudGoalManager{
		scienceGoals: make(map[string]*datatype.ScienceGoal),
	}, nil
}

// UpdateScienceGoal stores given science goal
func (cgm *CloudGoalManager) UpdateScienceGoal(scienceGoal *datatype.ScienceGoal) error {
	// TODO: This operation may need a mutex?
	cgm.scienceGoals[scienceGoal.ID] = scienceGoal
	return nil
}

// GetScienceGoal returns the science goal matching to given science goal ID
func (cgm *CloudGoalManager) GetScienceGoal(goalID string) (*datatype.ScienceGoal, error) {
	// TODO: This operation may need a mutex?
	if goal, exist := cgm.scienceGoals[goalID]; exist {
		return goal, nil
	}
	return nil, fmt.Errorf("the goal %s does not exist", goalID)
}

// GetScienceGoalsForNode returns a list of goals associated to given node
func (cgm *CloudGoalManager) GetScienceGoalsForNode(nodeName string) (goals []*datatype.ScienceGoal) {
	for _, scienceGoal := range cgm.scienceGoals {
		for _, subGoal := range scienceGoal.SubGoals {
			if subGoal.Node.Name == nodeName {
				goals = append(goals, scienceGoal)
			}
		}
	}
	return
}
