package cloudscheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

// CloudGoalManager structs a goal manager for cloudscheduler
type CloudGoalManager struct {
	scienceGoals map[string]*datatype.ScienceGoal
	rmqHandler   *interfacing.RabbitMQHandler
	Notifier     *interfacing.Notifier
	jobs         map[string]*datatype.Job
	mu           sync.Mutex
}

// SetRMQHandler sets a RabbitMQ handler used for transferring goals to edge schedulers
func (cgm *CloudGoalManager) SetRMQHandler(rmqHandler *interfacing.RabbitMQHandler) {
	cgm.rmqHandler = rmqHandler
	cgm.rmqHandler.CreateExchange("scheduler")
}

func (cgm *CloudGoalManager) AddJob(job *datatype.Job) string {
	newJobID := cgm.GenerateNewJobID()
	job.JobID = newJobID
	job.UpdateStatus(datatype.JobCreated)
	cgm.jobs[job.JobID] = job
	return newJobID
}

func (cgm *CloudGoalManager) GetJobs() map[string]*datatype.Job {
	return cgm.jobs
}

func (cgm *CloudGoalManager) GetJob(jobID string) (*datatype.Job, error) {
	if job, exist := cgm.jobs[jobID]; exist {
		return job, nil
	} else {
		return nil, fmt.Errorf("Job ID %q does not exist", jobID)
	}
}

func (cgm *CloudGoalManager) UpdateJob(job *datatype.Job, submit bool) {
	cgm.jobs[job.JobID] = job
	if submit {
		job.UpdateStatus(datatype.JobSubmitted)
		newScienceGoal := job.ScienceGoal
		cgm.UpdateScienceGoal(newScienceGoal)
		event := datatype.NewEventBuilder(datatype.EventGoalStatusNew).AddGoal(newScienceGoal).Build()
		cgm.Notifier.Notify(event)
	}
}

func (cgm *CloudGoalManager) SuspendJob(jobID string) error {
	if job, exist := cgm.jobs[jobID]; exist {
		job.UpdateStatus(datatype.JobSuspended)
		event := datatype.NewEventBuilder(datatype.EventJobStatusSuspended).
			AddJob(job).
			AddReason("Suspended by user").Build()
		cgm.Notifier.Notify(event)
		return nil
	} else {
		return fmt.Errorf("Failed to find job %q to suspend", jobID)
	}
}

func (cgm *CloudGoalManager) RemoveJob(jobID string, force bool) error {
	if job, exist := cgm.jobs[jobID]; exist {
		if job.Status == datatype.JobRunning && !force {
			return fmt.Errorf("Failed to remove job %q as it is in running state. Suspend it first or specify force=true", jobID)
		}
		delete(cgm.jobs, job.JobID)
		cgm.RemoveScienceGoal(job.ScienceGoal.ID)
		return nil
	} else {
		return fmt.Errorf("Failed to find job %q to remove", jobID)
	}
}

func (cgm *CloudGoalManager) RemoveScienceGoal(goalID string) error {
	if goal, exist := cgm.scienceGoals[goalID]; exist {
		delete(cgm.scienceGoals, goal.ID)
		return nil
	} else {
		return fmt.Errorf("Failed to find science goal %q to remove", goalID)
	}
}

// UpdateScienceGoal stores given science goal
func (cgm *CloudGoalManager) UpdateScienceGoal(scienceGoal *datatype.ScienceGoal) error {
	// TODO: This operation may need a mutex?
	cgm.scienceGoals[scienceGoal.ID] = scienceGoal

	// Send the updated science goal to all subject edge schedulers
	// if cgm.rmqHandler != nil {
	// 	// TODO: Refine what to send to RMQ for edge scheduler
	// 	// Send the updates
	// 	for _, subGoal := range scienceGoal.SubGoals {
	// 		message, err := yaml.Marshal([]*datatype.ScienceGoal{scienceGoal})
	// 		if err != nil {
	// 			logger.Error.Printf("Unable to parse the science goal <%s> into YAML: %s", scienceGoal.ID, err.Error())
	// 			continue
	// 		}
	// 		logger.Debug.Printf("%+v", string(message))
	// 		cgm.rmqHandler.SendYAML(subGoal.Name, message)
	// 	}
	// }

	return nil
}

// GetScienceGoal returns the science goal matching to given science goal ID
func (cgm *CloudGoalManager) GetScienceGoal(goalID string) (*datatype.ScienceGoal, error) {
	// TODO: This operation may need a mutex?
	if goal, exist := cgm.scienceGoals[goalID]; exist {
		return goal, nil
	}
	return nil, fmt.Errorf("Failed to find goal %q", goalID)
}

// GetScienceGoalsForNode returns a list of goals associated to given node
func (cgm *CloudGoalManager) GetScienceGoalsForNode(nodeName string) (goals []*datatype.ScienceGoal) {
	for _, scienceGoal := range cgm.scienceGoals {
		for _, subGoal := range scienceGoal.SubGoals {
			if strings.ToLower(subGoal.Name) == strings.ToLower(nodeName) {
				goals = append(goals, scienceGoal)
			}
		}
	}
	return
}

func (cgm *CloudGoalManager) GenerateNewJobID() string {
	cgm.mu.Lock()
	defer cgm.mu.Unlock()
	filePath := "/tmp/jobcounter"
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		ioutil.WriteFile(filePath, []byte("1"), 0600)
		return "1"
	} else {
		counter, err := ioutil.ReadFile(filePath)
		if err != nil {
			panic(err)
		}
		intCounter, _ := strconv.Atoi(string(counter))
		intCounter += 1
		ioutil.WriteFile(filePath, []byte(fmt.Sprint(intCounter)), 0600)
		return string(counter)
	}
}
