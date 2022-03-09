package cloudscheduler

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
)

// CloudGoalManager structs a goal manager for cloudscheduler
type CloudGoalManager struct {
	scienceGoals map[string]*datatype.ScienceGoal
	rmqHandler   *interfacing.RabbitMQHandler
	Notifier     *interfacing.Notifier
	jobs         map[string]*datatype.Job
}

// SetRMQHandler sets a RabbitMQ handler used for transferring goals to edge schedulers
func (cgm *CloudGoalManager) SetRMQHandler(rmqHandler *interfacing.RabbitMQHandler) {
	cgm.rmqHandler = rmqHandler
	cgm.rmqHandler.CreateExchange("scheduler")
}

func (cgm *CloudGoalManager) AddJob(job *datatype.Job) error {
	if _, exist := cgm.jobs[job.Name]; exist {
		return fmt.Errorf("Job already exists: %s", job.Name)
	}
	cgm.jobs[job.Name] = job
	return nil
}

func (cgm *CloudGoalManager) GetJobs() map[string]*datatype.Job {
	return cgm.jobs
}

// UpdateScienceGoal stores given science goal
func (cgm *CloudGoalManager) UpdateScienceGoal(scienceGoal *datatype.ScienceGoal) error {
	// TODO: This operation may need a mutex?
	cgm.scienceGoals[scienceGoal.ID] = scienceGoal

	// Send the updated science goal to all subject edge schedulers
	if cgm.rmqHandler != nil {
		// TODO: Refine what to send to RMQ for edge scheduler
		// Send the updates
		for _, subGoal := range scienceGoal.SubGoals {
			message, err := yaml.Marshal([]*datatype.ScienceGoal{scienceGoal})
			if err != nil {
				logger.Error.Printf("Unable to parse the science goal <%s> into YAML: %s", scienceGoal.ID, err.Error())
				continue
			}
			logger.Debug.Printf("%+v", string(message))
			cgm.rmqHandler.SendYAML(subGoal.Node.Name, message)
		}
	}

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
