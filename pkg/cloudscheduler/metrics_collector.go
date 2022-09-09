package cloudscheduler

type JobsMetric struct {
	CountSubmitted int
	CountRunning   int
	CountCompleted int
}

func (m *JobsMetric) GrantTotal() int {
	return m.CountSubmitted + m.CountRunning + m.CountCompleted
}
