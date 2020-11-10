package nodescheduler

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
)

var (
	scienceGoals          map[string]*datatype.ScienceGoal
	cloudSchedulerBaseURL string
	nodeID                string
)

// InitializeGoalManager manages goals received from the cloud scheduler
func InitializeGoalManager() {
	scienceGoals = make(map[string]*datatype.ScienceGoal)
	cloudSchedulerBaseURL = "http://localhost:9770"
	nodeID = "wb01"
}

// RunGoalManager keeps pulling science goals assigned to the node
// from the cloud scheduler
func RunGoalManager() {
	for {
		pullGoals()
		time.Sleep(3 * time.Second)
	}
}

func pullGoals() bool {
	queryURL, _ := url.Parse(cloudSchedulerBaseURL)
	queryURL.Path = path.Join(queryURL.Path, "api/v1/goals/", nodeID)
	logger.Info.Printf("%s", queryURL.String())
	resp, err := http.Get(queryURL.String())
	if err != nil {
		logger.Error.Printf("%s", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error.Printf("%s", err)
		return false
	}
	var pulledGoals []datatype.ScienceGoal
	_ = yaml.Unmarshal(body, &pulledGoals)

	logger.Info.Printf("%v", pulledGoals)

	triggerSchedule := false
	for _, goal := range pulledGoals {
		goalID := goal.ID
		if _, ok := scienceGoals[goalID]; !ok {
			scienceGoals[goalID] = &goal
			triggerSchedule = true
		}
	}
	if triggerSchedule {
		chanTriggerSchedule <- "received new goals from SES"
	}
	return true
}
