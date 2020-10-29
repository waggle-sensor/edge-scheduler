package jobmanager

var (
	chanToJobManager chan *ScienceGoal
	scienceGoals     map[string]*ScienceGoal
)

func InitializeJobManager() {
	chanToJobManager = make(chan *ScienceGoal)
	scienceGoals = make(map[string]*ScienceGoal)
}

func RunJobManager() {
	for {
		scienceGoal := <-chanToJobManager
		scienceGoals[scienceGoal.ID] = scienceGoal
	}
}
