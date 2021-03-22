package nodescheduler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
)

// GoalManager structs a goal manager for nodescheduler
type NodeGoalManager struct {
	scienceGoals             map[string]*datatype.ScienceGoal
	cloudSchedulerBaseURL    string
	NodeID                   string
	chanNewGoalToGoalManager chan *datatype.ScienceGoal
}

// NewGoalManager creates and returns an instance of goal manager
func NewNodeGoalManager(cloudSchedulerURL string, nodeID string) (*NodeGoalManager, error) {
	return &NodeGoalManager{
		scienceGoals:             make(map[string]*datatype.ScienceGoal),
		cloudSchedulerBaseURL:    cloudSchedulerURL,
		NodeID:                   nodeID,
		chanNewGoalToGoalManager: make(chan *datatype.ScienceGoal, 100),
	}, nil
}

// GetScienceGoal returns the goal of given goal_id
func (ngm *NodeGoalManager) GetScienceGoal(goalID string) (*datatype.ScienceGoal, error) {
	if goal, exist := ngm.scienceGoals[goalID]; exist {
		return goal, nil
	}
	return nil, fmt.Errorf("The goal %s does not exist", goalID)
}

// RunGoalManager handles goal related events from both cloud and local
// and keeps goals managed up-to-date with the help from the events
func (ngm *NodeGoalManager) Run(chanToScheduler chan *datatype.ScienceGoal) {
	go ngm.pullingGoalsFromCloudScheduler()

	for {
		select {
		case scienceGoal := <-ngm.chanNewGoalToGoalManager:
			logger.Info.Printf("Received a goal from SES:%s id:(%s)", scienceGoal.Name, scienceGoal.ID)
			ngm.scienceGoals[scienceGoal.ID] = scienceGoal
			chanToScheduler <- scienceGoal
		}
	}
}

// pullingGoalsFromCloudScheduler periodically pulls goals from the cloud scheduler
func (gm *NodeGoalManager) pullingGoalsFromCloudScheduler() {
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
		for _, goal := range pulledGoals {
			if _, exist := gm.scienceGoals[goal.ID]; !exist {
				gm.chanNewGoalToGoalManager <- &goal
			}
		}
	}
}
