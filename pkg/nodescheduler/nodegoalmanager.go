package nodescheduler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/watch"
)

// GoalManager structs a goal manager for nodescheduler
type NodeGoalManager struct {
	ScienceGoals          map[string]*datatype.ScienceGoal
	cloudSchedulerBaseURL string
	NodeID                string
	Notifier              *interfacing.Notifier
	Simulate              bool
	chanGoalQueue         chan *datatype.ScienceGoal
	rmqHandler            *interfacing.RabbitMQHandler
	GoalWatcher           watch.Interface
}

// NewGoalManager creates and returns an instance of goal manager
func NewNodeGoalManager(cloudSchedulerURL string, nodeID string, simulate bool) (*NodeGoalManager, error) {
	return &NodeGoalManager{
		ScienceGoals:          make(map[string]*datatype.ScienceGoal),
		cloudSchedulerBaseURL: cloudSchedulerURL,
		NodeID:                nodeID,
		Simulate:              simulate,
		chanGoalQueue:         make(chan *datatype.ScienceGoal, 100),
	}, nil
}

// GetScienceGoalByID returns the goal of given goal name
func (ngm *NodeGoalManager) GetScienceGoalByID(goalID string) (*datatype.ScienceGoal, error) {
	if goal, exist := ngm.ScienceGoals[goalID]; exist {
		return goal, nil
	}

	return nil, fmt.Errorf("The goal ID %s does not exist", goalID)
}

func (ngm *NodeGoalManager) GetScienceGoalByName(goalName string) (*datatype.ScienceGoal, error) {
	for _, goal := range ngm.ScienceGoals {
		if goal.Name == goalName {
			return goal, nil
		}
	}
	return nil, fmt.Errorf("The goal Name %s does not exist", goalName)
}

// SetRMQHandler sets a RabbitMQ handler used for transferring goals to edge schedulers
func (ngm *NodeGoalManager) SetRMQHandler(rmqHandler *interfacing.RabbitMQHandler) {
	ngm.rmqHandler = rmqHandler
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
	if g, exist := ngm.ScienceGoals[goalID]; exist {
		delete(ngm.ScienceGoals, goalID)
		ngm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventGoalStatusDeleted).AddGoal(g).Build())
		return nil
	} else {
		return fmt.Errorf("Failed to find goal by ID %s", goalID)
	}
}

func (ngm *NodeGoalManager) AddGoal(goal *datatype.ScienceGoal) {
	ngm.chanGoalQueue <- goal
}

// SetGoals addes given science goals and drops existing goals.
//
// If a goal already exists and content of the goal is the same with the existing goal, it skips.
func (ngm *NodeGoalManager) SetGoals(goals [](datatype.ScienceGoal)) {
	goalIDs := make(map[string]bool)
	for _, goal := range goals {
		goalIDs[goal.ID] = true
	}
	// Remove existing goals not included in the new goals
	for _, goal := range ngm.ScienceGoals {
		if _, exist := goalIDs[goal.ID]; !exist {
			ngm.DropGoal(goal.ID)
		}
	}
	for _, goal := range goals {
		if subGoal := goal.GetMySubGoal(ngm.NodeID); subGoal != nil {
			subGoal.AddChecksum()
			ngm.AddGoal(&goal)
		}
	}
}

// RunGoalManager handles goal related events from both cloud and local
// and keeps goals managed up-to-date with the help from the events
func (ngm *NodeGoalManager) Run(chanToScheduler chan datatype.Event) {
	// NOTE: use RabbitMQ to receive goals if set
	// var useRabbitMQ bool
	// if ngm.rmqHandler != nil {
	// 	useRabbitMQ = true
	// } else {
	// 	useRabbitMQ = false
	// }
	if !ngm.Simulate {
		// go ngm.pullGoalsFromCloudScheduler(useRabbitMQ)
	}
	for {
		select {
		case scienceGoal := <-ngm.chanGoalQueue:
			logger.Debug.Printf("Received a goal %q", scienceGoal.Name)
			if goal, err := ngm.GetScienceGoalByName(scienceGoal.Name); err == nil {
				// if goal.GetMySubGoal(ngm.NodeID) == scienceGoal.GetMySubGoal(ngm.NodeID) {
				if goal.GetMySubGoal(ngm.NodeID).CompareChecksum(scienceGoal.GetMySubGoal(ngm.NodeID)) {
					logger.Debug.Printf("The newly submitted goal %s exists and no changes in the goal. Skipping adding the goal", scienceGoal.Name)
				} else {
					logger.Debug.Printf("The newly submitted goal %s exists and has changed its content. Need scheduling", scienceGoal.Name)
					ngm.ScienceGoals[scienceGoal.ID] = scienceGoal
					ngm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventGoalStatusUpdated).AddGoal(scienceGoal).Build())
				}
			} else {
				ngm.ScienceGoals[scienceGoal.ID] = scienceGoal
				ngm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventGoalStatusReceived).AddGoal(scienceGoal).Build())
			}
		}
	}
}

// pullingGoalsFromCloudScheduler periodically pulls goals from the cloud scheduler
func (gm *NodeGoalManager) pullGoalsFromCloudScheduler(useRabbitMQ bool) {
	if useRabbitMQ {
		for {
			logger.Info.Printf("SES endpoint: %s", gm.rmqHandler.RabbitmqURI)
			gm.rmqHandler.DeclareQueueAndConnectToExchange("scheduler", gm.NodeID)
			msgs, err := gm.rmqHandler.GetReceiver(gm.NodeID)
			if err != nil {
				logger.Error.Printf("%s", err)
				time.Sleep(5 * time.Second)
				continue
			}
			for d := range msgs {
				var pulledGoals []datatype.ScienceGoal
				err = yaml.Unmarshal(d.Body, &pulledGoals)
				if err != nil {
					logger.Error.Printf("%s", err)
				}
				logger.Info.Printf("%v", pulledGoals)
				// TODO: this does not account for goal status change
				//       from SES -- may or may not happen
				// for _, goal := range pulledGoals {
				// 	if _, exist := gm.scienceGoals[goal.ID]; !exist {
				// 		gm.chanNewGoalToGoalManager <- &goal
				// 	}
				// }
			}
		}
	} else {
		queryURL, _ := url.Parse(gm.cloudSchedulerBaseURL)
		queryURL.Path = path.Join(queryURL.Path, "api/v1/goals/", gm.NodeID)
		logger.Info.Printf("SES endpoint: %s", queryURL.String())
		for {
			time.Sleep(5 * time.Second)
			resp, err := http.Get(queryURL.String())
			if err != nil {
				logger.Error.Printf("%s", err)
				continue
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.Error.Printf("%s", err)
				continue
			}
			var pulledGoals []datatype.ScienceGoal
			_ = yaml.Unmarshal(body, &pulledGoals)

			// TODO: this does not account for goal status change
			//       from SES -- may or may not happen
			// for _, goal := range pulledGoals {
			// 	if _, exist := gm.scienceGoals[goal.ID]; !exist {
			// 		gm.chanNewGoalToGoalManager <- &goal
			// 	}
			// }
		}
	}
}
