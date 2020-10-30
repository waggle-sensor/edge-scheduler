package cloudscheduler

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
)

var (
	chanToJobManager chan *datatype.ScienceGoal
	scienceGoals     map[string]*datatype.ScienceGoal
)

// InitializeJobManager initializes job manager
func InitializeJobManager() {
	chanToJobManager = make(chan *datatype.ScienceGoal)
	scienceGoals = make(map[string]*datatype.ScienceGoal)
}

// RunJobManager is a goroutine that manages science goals
func RunJobManager() {
	for {
		scienceGoal := <-chanToJobManager
		scienceGoals[scienceGoal.ID] = scienceGoal
	}
}
