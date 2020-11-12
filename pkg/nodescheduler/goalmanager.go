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
	chanToGoalManager     = make(chan Event)
)

// InitializeGoalManager manages goals received from the cloud scheduler
func InitializeGoalManager() {
	scienceGoals = make(map[string]*datatype.ScienceGoal)
	cloudSchedulerBaseURL = "http://localhost:9770"
	nodeID = "wb01"
}

// RunGoalManager handles goal related events from both cloud and local
// and keeps goals managed up-to-date with the help from the events
func RunGoalManager() {
	go pullingGoals()
	for {

	}
}

func pullingGoals() {
	queryURL, _ := url.Parse(cloudSchedulerBaseURL)
	queryURL.Path = path.Join(queryURL.Path, "api/v1/goals/", nodeID)
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
			if _, exist := scienceGoals[goal.ID]; !exist {
				chanToGoalManager <- &goal
			}
		}
	}
}
