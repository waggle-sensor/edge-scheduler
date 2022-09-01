package cloudscheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	yaml "gopkg.in/yaml.v2"
	// "github.com/urfave/negroni"
)

type APIServer struct {
	version                string
	port                   int
	enablePushNotification bool
	mainRouter             *mux.Router
	cloudScheduler         *CloudScheduler
	subscribers            map[string]map[chan *datatype.Event]bool
}

func (api *APIServer) subscribe(nodeName string, c chan *datatype.Event) {
	nodeName = strings.ToLower(nodeName)
	if _, exist := api.subscribers[nodeName]; !exist {
		api.subscribers[nodeName] = make(map[chan *datatype.Event]bool)
	}
	api.subscribers[nodeName][c] = true
}

func (api *APIServer) unsubscribe(nodeName string, c chan *datatype.Event) {
	nodeName = strings.ToLower(nodeName)
	if _, exist := api.subscribers[nodeName]; exist {
		delete(api.subscribers[nodeName], c)
	}
}

func (api *APIServer) Push(nodeName string, event *datatype.Event) {
	nodeName = strings.ToLower(nodeName)
	if _, exist := api.subscribers[nodeName]; exist {
		for ch := range api.subscribers[nodeName] {
			select {
			case ch <- event:
			default:
				// (Sean) don't block on slow channels. assume they will drop and reconnect to fetch goal.
			}
		}
	}
}

func (api *APIServer) Run() {
	api_address_port := fmt.Sprintf("0.0.0.0:%d", api.port)
	logger.Info.Printf("API server starts at %q...", api_address_port)
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := datatype.NewAPIMessageBuilder().
			AddEntity("id", fmt.Sprintf("Cloud Scheduler (%s)", api.cloudScheduler.Name)).
			AddEntity("version", api.version).Build()
		respondJSON(w, http.StatusOK, response.ToJson())
		fmt.Fprintln(w)
	})
	api_route := r.PathPrefix("/api/v1").Subrouter()
	api_route.Handle("/create", http.HandlerFunc(api.handlerCreateJob)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle("/edit", http.HandlerFunc(api.handlerEditJob)).Methods(http.MethodPost)
	api_route.Handle("/submit", http.HandlerFunc(api.handlerSubmitJobs)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle("/jobs", http.HandlerFunc(api.handlerJobs)).Methods(http.MethodGet)
	api_route.Handle("/jobs/{id}/status", http.HandlerFunc(api.handlerJobStatus)).Methods(http.MethodGet)
	api_route.Handle("/jobs/{id}/rm", http.HandlerFunc(api.handlerJobRemove)).Methods(http.MethodGet)
	// api.Handle("/goals", http.HandlerFunc(cs.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	api_route.Handle("/goals/{nodeName}", http.HandlerFunc(api.handlerGoalForNode)).Methods(http.MethodGet)
	if api.enablePushNotification {
		logger.Info.Printf("Enabling push notification. Nodes can connect to /goals/{nodeName}/stream to get notification from the cloud scheduler.")
		api_route.Handle("/goals/{nodeName}/stream", http.HandlerFunc(api.handlerGoalStreamForNode)).Methods(http.MethodGet)
	}
	logger.Info.Fatalln(http.ListenAndServe(api_address_port, api.mainRouter))
}

func (api *APIServer) handlerCreateJob(w http.ResponseWriter, r *http.Request) {
	var newJob *datatype.Job
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		if _, exist := queries["name"]; !exist {
			response := datatype.NewAPIMessageBuilder().AddError("name field is required").Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
		name := queries.Get("name")
		newJob = &datatype.Job{
			Name: name,
		}
	case http.MethodPost:
		// The query includes a full job description
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		} else {
			err = yaml.Unmarshal(blob, &newJob)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
		}
	}
	jobID := api.cloudScheduler.GoalManager.AddJob(newJob)
	response := datatype.NewAPIMessageBuilder().
		AddEntity("job_name", newJob.Name).
		AddEntity("job_id", jobID).
		AddEntity("status", datatype.JobCreated).
		Build()
	respondJSON(w, http.StatusOK, response.ToJson())
}

func (api *APIServer) handlerEditJob(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	if _, exist := queries["id"]; exist {
		jobID := queries.Get("id")
		_, err := api.cloudScheduler.GoalManager.GetJob(jobID)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
		// TODO: the API always assumes that the body contains job content
		updatedJob := datatype.NewJob("", "", "")
		// The query includes a full job description
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		} else {
			err = yaml.Unmarshal(blob, &updatedJob)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
			updatedJob.JobID = jobID
			api.cloudScheduler.GoalManager.UpdateJob(updatedJob, false)
			response := datatype.NewAPIMessageBuilder().AddEntity("job_id", jobID).AddEntity("status", datatype.JobDrafted)
			respondJSON(w, http.StatusOK, response.Build().ToJson())
			return
		}
	} else {
		response := datatype.NewAPIMessageBuilder().AddError("job_id is required").Build()
		respondJSON(w, http.StatusBadRequest, response.ToJson())
		return
	}
}

func (api *APIServer) handlerSubmitJobs(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	flagDryRun := false
	if _, exist := queries["dryrun"]; exist {
		f, err := strconv.ParseBool(queries.Get("dryrun"))
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
		flagDryRun = f
	}
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		if _, exist := queries["id"]; exist {
			errorList := api.cloudScheduler.ValidateJobAndCreateScienceGoal(queries.Get("id"), flagDryRun)
			if len(errorList) > 0 {
				response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("%v", errorList)).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			} else {
				response := datatype.NewAPIMessageBuilder().AddEntity("job_id", queries.Get("id"))
				if flagDryRun {
					response = response.AddEntity("dryrun", true)
				} else {
					response = response.AddEntity("status", datatype.JobSubmitted)
				}
				respondJSON(w, http.StatusOK, response.Build().ToJson())
				return
			}
		} else {
			response := datatype.NewAPIMessageBuilder().AddError("job_id is required").Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
	case http.MethodPost:
		newJob := datatype.NewJob("", "", "")
		// The query includes a full job description
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		} else {
			err = yaml.Unmarshal(blob, &newJob)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
			jobID := api.cloudScheduler.GoalManager.AddJob(newJob)
			errorList := api.cloudScheduler.ValidateJobAndCreateScienceGoal(jobID, flagDryRun)
			if len(errorList) > 0 {
				response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("%v", errorList)).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			} else {
				response := datatype.NewAPIMessageBuilder().AddEntity("job_id", jobID)
				if flagDryRun {
					response = response.AddEntity("dryrun", true)
				} else {
					response = response.AddEntity("status", datatype.JobSubmitted)
				}
				respondJSON(w, http.StatusOK, response.Build().ToJson())
				return
			}
		}
	}

	// switch r.Method {
	// case PUT, POST:
	// 	data, err := ioutil.ReadAll(r.Body)
	// 	if err != nil {
	// 		respondJSON(w, http.StatusBadRequest, err.Error())
	// 		return
	// 	}
	// 	var job datatype.Job
	// 	err = json.Unmarshal(data, &job)
	// 	if err != nil {
	// 		respondJSON(w, http.StatusBadRequest, err.Error())
	// 		return
	// 	}
	// 	err = api.cloudScheduler.GoalManager.AddJob(&job)
	// 	if err != nil {
	// 		respondJSON(w, http.StatusBadRequest, err.Error())
	// 	}
	// 	respondJSON(w, http.StatusOK, `{"response": "success"}`)
	// 	return
	// }
	// respondJSON(w, http.StatusOK, []byte{})
	// if r.Method == POST {
	// 	log.Printf("hit POST")
	// 	// yamlFile, err := ioutil.ReadAll(r.Body)
	// 	// if err != nil {
	// 	// 	fmt.Println(err)
	// 	// }
	// 	// var job datatype.Job
	// 	// _ = yaml.Unmarshal(yamlFile, &job)
	// 	// job.ID = guuid.New().String()

	// 	// if len(job.PluginTags) > 0 {
	// 	// 	foundPlugins := cs.Meta.GetPluginsByTags(job.PluginTags)
	// 	// 	for _, p := range foundPlugins {
	// 	// 		logger.Debug.Printf("Plugin %s:%s is added to job %s", p.Name, p.PluginSpec.Version, job.Name)
	// 	// 		job.AddPlugin(p)
	// 	// 	}
	// 	// 	logger.Info.Printf("Found %d plugins by the tags", len(foundPlugins))
	// 	// }

	// 	// if len(job.NodeTags) > 0 {
	// 	// 	foundNodes := cs.Meta.GetNodesByTags(job.NodeTags)
	// 	// 	for _, n := range foundNodes {
	// 	// 		logger.Debug.Printf("Node %s is added to job %s", n.Name, job.Name)
	// 	// 		job.AddNode(n)
	// 	// 	}
	// 	// 	logger.Info.Printf("Found %d nodes by the tags", len(foundNodes))
	// 	// }

	// 	// // TODO: Add error hanlding here
	// 	// scienceGoal, errorList := cs.Validator.ValidateJobAndCreateScienceGoal(&job, cs.Meta)
	// 	// if len(errorList) > 0 {
	// 	// 	for _, err := range errorList {
	// 	// 		logger.Error.Printf("%s", err)
	// 	// 	}
	// 	// } else {
	// 	// 	cs.GoalManager.UpdateScienceGoal(scienceGoal)
	// 	// }
	// 	// respondYAML(w, http.StatusOK, scienceGoal)

	// 	respondJSON(w, http.StatusNotFound, "Not supported yet")
	// } else if r.Method == PUT {
	// 	log.Printf("hit PUT")
	// 	// mReader, err := r.MultipartReader()
	// 	// if err != nil {
	// 	// 	respondJSON(w, http.StatusOK, "ERROR")
	// 	// }
	// 	// yamlFile, err := ioutil.ReadAll(r.Body)
	// 	// if err != nil {
	// 	// 	fmt.Println(err)
	// 	// }
	// 	// var job datatype.Job
	// 	// _ = yaml.Unmarshal(yamlFile, &job)
	// 	// job.ID = guuid.New().String()

	// 	// if len(job.PluginTags) > 0 {
	// 	// 	foundPlugins := cs.Meta.GetPluginsByTags(job.PluginTags)
	// 	// 	for _, p := range foundPlugins {
	// 	// 		logger.Debug.Printf("Plugin %s:%s is added to job %s", p.Name, p.PluginSpec.Version, job.Name)
	// 	// 		job.AddPlugin(p)
	// 	// 	}
	// 	// 	logger.Info.Printf("Found %d plugins by the tags", len(foundPlugins))
	// 	// }

	// 	// if len(job.NodeTags) > 0 {
	// 	// 	foundNodes := cs.Meta.GetNodesByTags(job.NodeTags)
	// 	// 	for _, n := range foundNodes {
	// 	// 		logger.Debug.Printf("Node %s is added to job %s", n.Name, job.Name)
	// 	// 		job.AddNode(n)
	// 	// 	}
	// 	// 	logger.Info.Printf("Found %d nodes by the tags", len(foundNodes))
	// 	// }

	// 	// // TODO: Add error hanlding here
	// 	// scienceGoal, errorList := cs.Validator.ValidateJobAndCreateScienceGoal(&job, cs.Meta)
	// 	// if len(errorList) > 0 {
	// 	// 	for _, err := range errorList {
	// 	// 		logger.Error.Printf("%s", err)
	// 	// 	}
	// 	// } else {
	// 	// 	cs.GoalManager.UpdateScienceGoal(scienceGoal)
	// 	// }
	// 	// respondYAML(w, http.StatusOK, scienceGoal)
	// 	respondJSON(w, http.StatusOK, "")
	// }
}

func (api *APIServer) handlerJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		jobs := api.cloudScheduler.GoalManager.GetJobs()
		response := datatype.NewAPIMessageBuilder()
		for _, job := range jobs {
			response.AddEntity(job.JobID, job)
		}
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	}
}

func (api *APIServer) handlerJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if r.Method == http.MethodGet {
		logger.Debug.Printf("API call on Job status of %s", vars["id"])
		// if goal, err := cs.GoalManager.GetScienceGoal(vars["id"]); err == nil {
		// 	respondJSON(w, http.StatusOK, goal)
		// } else {
		// 	respondJSON(w, http.StatusOK, "")
		// }
		job, err := api.cloudScheduler.GoalManager.GetJob(vars["id"])
		response := datatype.NewAPIMessageBuilder()
		if err != nil {
			response.AddError(err.Error())
		} else {
			response.AddEntity(vars["id"], job)
		}
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	}
}

func (api *APIServer) handlerJobRemove(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	jobID := queries.Get("id")
	if _, exist := queries["suspend"]; exist {
		suspend := queries.Get("suspend")
		if suspend == "true" {
			err := api.cloudScheduler.GoalManager.SuspendJob(jobID)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddEntity("job_id", jobID).
					AddError(err.Error()).Build()
				respondJSON(w, http.StatusOK, response.ToJson())
				return
			}
			response := datatype.NewAPIMessageBuilder().AddEntity("job_id", jobID).
				AddEntity("status", datatype.JobSuspended).Build()
			respondJSON(w, http.StatusOK, response.ToJson())
			return
		}
	}
	force := false
	if _, exist := queries["force"]; exist {
		forceString := queries.Get("force")
		if forceString == "true" {
			force = true
		}
	}
	err := api.cloudScheduler.GoalManager.RemoveJob(jobID, force)
	if err != nil {
		response := datatype.NewAPIMessageBuilder().AddEntity("job_id", jobID).
			AddError(err.Error()).Build()
		respondJSON(w, http.StatusOK, response.ToJson())
	} else {
		response := datatype.NewAPIMessageBuilder().
			AddEntity("job_id", jobID).
			AddEntity("status", datatype.JobRemoved).Build()
		respondJSON(w, http.StatusOK, response.ToJson())
	}
}

func (api *APIServer) handlerGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {

	} else if r.Method == http.MethodPost {
		log.Printf("hit POST")

		respondJSON(w, http.StatusOK, []byte{})
	} else if r.Method == http.MethodPut {
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
		respondJSON(w, http.StatusOK, []byte{})
	}
}

func (api *APIServer) handlerGoalForNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]
	var goals []*datatype.ScienceGoal
	for _, g := range api.cloudScheduler.GoalManager.GetScienceGoalsForNode(nodeName) {
		goals = append(goals, g.ShowMyScienceGoal(nodeName))
	}
	blob, err := json.MarshalIndent(goals, "", "  ")
	if err != nil {
		response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
		respondJSON(w, http.StatusOK, response.ToJson())
		return
	}
	respondJSON(w, http.StatusOK, blob)
}

// handlerGoalStreamForNode uses server-sent events (SSE) to stream new goals to connected nodes
// whenever goals are changed in cloud scheduler
func (api *APIServer) handlerGoalStreamForNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/event-stream")
	c := make(chan *datatype.Event, 1)
	api.subscribe(nodeName, c)
	defer api.unsubscribe(nodeName, c)
	var goals []*datatype.ScienceGoal
	for _, g := range api.cloudScheduler.GoalManager.GetScienceGoalsForNode(nodeName) {
		goals = append(goals, g.ShowMyScienceGoal(nodeName))
	}
	// if no science goal is assigned to the node return an empty list []
	// returning null may raise an exception in edge scheduler
	if len(goals) > 0 {
		blob, err := json.MarshalIndent(goals, "", "  ")
		if err != nil {
			logger.Error.Printf("Failed to compress goals for node %q before pushing", nodeName)
		} else {
			event := datatype.NewEventBuilder(datatype.EventGoalStatusUpdated).AddEntry("goals", string(blob)).Build()
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.ToString(), event.GetEntry("goals")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
	for event := range c {
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.ToString(), event.GetEntry("goals")); err != nil {
			return
		}
		flusher.Flush()
	}
}

func respondJSON(w http.ResponseWriter, statusCode int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
	// fmt.Fprintln(w, data)
	// json.NewEncoder(w).Encode(data)
	// s, err := json.MarshalIndent(data, "", "  ")
	// if err == nil {
	// 	w.Write(s)
	// }
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
