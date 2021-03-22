package cloudscheduler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
)

const (
	GET  = "GET"
	POST = "POST"
	PUT  = "PUT"
)

// CloudScheduler structs the cloud scheduler
type CloudScheduler struct {
	GoalManager *CloudGoalManager
	Meta        *MetaHandler
	Validator   *JobValidator
	mainRouter  *mux.Router
}

// NewCloudScheduler returns an instance of cloud scheduler
func NewCloudScheduler(mh *MetaHandler) (*CloudScheduler, error) {
	gm, err := NewCloudGoalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloud goal manager: %s", err.Error())
	}

	jv, err := NewJobValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize job validator: %s", err.Error())
	}

	return &CloudScheduler{
		GoalManager: gm,
		Meta:        mh,
		Validator:   jv,
		mainRouter:  mux.NewRouter(),
	}, nil
}

func (cs *CloudScheduler) Run(csName string, port int) {
	cs.mainRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprint("`{id: Cloud scheduler: ", csName, "}`")
		fmt.Fprintln(w, response)
	})

	api := cs.mainRouter.PathPrefix("/api/v1").Subrouter()
	api.Handle("/submit", http.HandlerFunc(cs.handlerSubmitJobs)).Methods(http.MethodPost, http.MethodPut)
	api.Handle("/jobs", http.HandlerFunc(cs.handlerJobs)).Methods(http.MethodGet)
	api.Handle("/jobs/{id}/status", http.HandlerFunc(cs.handlerJobStatus)).Methods(http.MethodGet)
	api.Handle("/goals", http.HandlerFunc(cs.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	api.Handle("/goals/{nodeName}", http.HandlerFunc(cs.handlerGoalForNode)).Methods(http.MethodGet)

	logger.Info.Printf("%s starts listening to port %d...", csName, port)
	logger.Info.Fatalln(http.ListenAndServe(fmt.Sprint("0.0.0.0:", port), cs.mainRouter))
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// json.NewEncoder(w).Encode(data)
	s, err := json.MarshalIndent(data, "", "  ")
	if err == nil {
		w.Write(s)
	}
}

func respondYAML(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(statusCode)

	// json.NewEncoder(w).Encode(data)
	s, err := yaml.Marshal(data)
	if err == nil {
		w.Write(s)
	}
}

func (cs *CloudScheduler) handlerSubmitJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == POST {
		log.Printf("hit POST")

		respondJSON(w, http.StatusNotFound, "Not supported yet")
	} else if r.Method == PUT {
		log.Printf("hit PUT")
		// mReader, err := r.MultipartReader()
		// if err != nil {
		// 	respondJSON(w, http.StatusOK, "ERROR")
		// }
		yamlFile, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
		}
		var job datatype.Job
		_ = yaml.Unmarshal(yamlFile, &job)
		job.ID = guuid.New().String()

		if len(job.PluginTags) > 0 {
			foundPlugins := cs.Meta.GetPluginsByTags(job.PluginTags)
			for _, p := range foundPlugins {
				logger.Debug.Printf("Plugin %s:%s is added to job %s", p.Name, p.Version, job.Name)
				job.AddPlugin(p)
			}
			logger.Info.Printf("Found %d plugins by the tags", len(foundPlugins))
		}

		if len(job.NodeTags) > 0 {
			foundNodes := cs.Meta.GetNodesByTags(job.NodeTags)
			for _, n := range foundNodes {
				logger.Debug.Printf("Node %s is added to job %s", n.Name, job.Name)
				job.AddNode(n)
			}
			logger.Info.Printf("Found %d nodes by the tags", len(foundNodes))
		}

		// TODO: Add error hanlding here
		scienceGoal, errorList := cs.Validator.ValidateJobAndCreateScienceGoal(&job, cs.Meta)
		if len(errorList) > 0 {
			for _, err := range errorList {
				logger.Error.Printf("%s", err)
			}
		} else {
			cs.GoalManager.UpdateScienceGoal(scienceGoal)
		}
		respondYAML(w, http.StatusOK, scienceGoal)
		// respondJSON(w, http.StatusOK, "")
	}
}

func (cs *CloudScheduler) handlerJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		log.Printf("hit GET")

		respondJSON(w, http.StatusOK, "")
	}
}

func (cs *CloudScheduler) handlerJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if r.Method == GET {
		log.Printf("hit GET")
		logger.Info.Printf("Job status of %s", vars["id"])
		if goal, err := cs.GoalManager.GetScienceGoal(vars["id"]); err == nil {
			respondJSON(w, http.StatusOK, goal)
		} else {
			respondJSON(w, http.StatusOK, "")
		}
	}
}

func (cs *CloudScheduler) handlerGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {

	} else if r.Method == POST {
		log.Printf("hit POST")

		respondJSON(w, http.StatusOK, "")
	} else if r.Method == PUT {
		log.Printf("hit PUT")
		// mReader, err := r.MultipartReader()
		// if err != nil {
		// 	respondJSON(w, http.StatusOK, "ERROR")
		// }
		// yamlFile, err := ioutil.ReadAll(r.Body)
		// if err != nil {
		// 	fmt.Println(err)
		// }
		// var goal Goal
		// _ = yaml.Unmarshal(yamlFile, &goal)
		// log.Printf("%v", goal)
		// RegisterGoal(goal)
		// chanTriggerScheduler <- "api server"
		respondJSON(w, http.StatusOK, "")
	}
}

func (cs *CloudScheduler) handlerGoalForNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if r.Method == GET {
		nodeName := vars["nodeName"]
		goals := cs.GoalManager.GetScienceGoalsForNode(nodeName)
		// dat, _ := yaml.Marshal(goals)
		respondYAML(w, http.StatusOK, goals)
		// respondYAML(w, http.StatusOK, `[{"response": "No goals found"}]`)
	}
}
