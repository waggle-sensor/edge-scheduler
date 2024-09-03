package cloudscheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	yaml "gopkg.in/yaml.v2"
	// "github.com/urfave/negroni"
)

const (
	API_V1_VERSION                             = "/api/v1"
	API_PATH_JOB_CREATE                        = "/create"
	API_PATH_JOB_EDIT                          = "/edit"
	API_PATH_JOB_SUBMIT                        = "/submit"
	API_PATH_JOB_LIST                          = "/jobs/list"
	API_PATH_JOB_STATUS_REGEX                  = "/jobs/%s/status"
	API_PATH_JOB_REMOVE_REGEX                  = "/jobs/%s/rm"
	API_PATH_JOB_TEMPLATE_REGEX                = "/jobs/%s/template"
	API_PATH_GOALS_NODE_REGEX                  = "/goals/%s"
	API_PATH_GOALS_NODE_STREAM_REGEX           = "/goals/%s/stream"
	MANAGEMENT_API_PATH_SYSTEM_METRICS         = "/system/metrics"
	MANAGEMENT_API_PATH_DB                     = "/db"
	MANAGEMENT_API_PATH_DATA_JOBS              = "/data/jobs"
	MANAGEMENT_API_PATH_DATA_PLUGINS           = "/data/plugins"
	MANAGEMENT_API_PATH_DATA_PLUGINS_WHITELIST = "/data/plugins/whitelist"
	MANAGEMENT_API_PATH_DATA_NODES             = "/data/nodes"
)

type APIServer struct {
	version                string
	port                   int
	managementPort         int
	enablePushNotification bool
	apiRouter              *mux.Router
	managementRouter       *mux.Router
	cloudScheduler         *CloudScheduler
	subscribers            map[string]map[chan datatype.Event]bool
	subscriberMutex        sync.Mutex
	authenticator          Authenticator
}

func (api *APIServer) subscribe(nodeName string, c chan datatype.Event) {
	nodeName = strings.ToLower(nodeName)
	api.subscriberMutex.Lock()
	if _, exist := api.subscribers[nodeName]; !exist {
		api.subscribers[nodeName] = make(map[chan datatype.Event]bool)
	}
	api.subscribers[nodeName][c] = true
	api.subscriberMutex.Unlock()
}

func (api *APIServer) unsubscribe(nodeName string, c chan datatype.Event) {
	nodeName = strings.ToLower(nodeName)
	api.subscriberMutex.Lock()
	if _, exist := api.subscribers[nodeName]; exist {
		delete(api.subscribers[nodeName], c)
	}
	api.subscriberMutex.Unlock()
}

func (api *APIServer) Push(nodeName string, event datatype.Event) {
	nodeName = strings.ToLower(nodeName)
	api.subscriberMutex.Lock()
	if _, exist := api.subscribers[nodeName]; exist {
		for ch := range api.subscribers[nodeName] {
			select {
			case ch <- event:
			default:
				// (Sean) don't block on slow channels. assume they will drop and reconnect to fetch goal.
			}
		}
	}
	api.subscriberMutex.Unlock()
}

func (api *APIServer) ConfigureAPIs(prometheusGatherer *prometheus.Registry) {
	api.apiRouter = mux.NewRouter()
	r := api.apiRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := datatype.NewAPIMessageBuilder().
			AddEntity("id", fmt.Sprintf("Cloud Scheduler (%s)", api.cloudScheduler.Name)).
			AddEntity("version", api.version).Build()
		respondJSON(w, http.StatusOK, response.ToJson())
		fmt.Fprintln(w)
	})
	api_route := r.PathPrefix(API_V1_VERSION).Subrouter()
	api_route.Handle(API_PATH_JOB_CREATE, http.HandlerFunc(api.handlerCreateJob)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle(API_PATH_JOB_EDIT, http.HandlerFunc(api.handlerEditJob)).Methods(http.MethodPost)
	api_route.Handle(API_PATH_JOB_SUBMIT, http.HandlerFunc(api.handlerSubmitJobs)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle(API_PATH_JOB_LIST, http.HandlerFunc(api.handlerJobs)).Methods(http.MethodGet)
	api_route.Handle(fmt.Sprintf(API_PATH_JOB_STATUS_REGEX, "{id}"), http.HandlerFunc(api.handlerJobStatus)).Methods(http.MethodGet)
	api_route.Handle(fmt.Sprintf(API_PATH_JOB_REMOVE_REGEX, "{id}"), http.HandlerFunc(api.handlerJobRemove)).Methods(http.MethodGet)
	api_route.Handle(fmt.Sprintf(API_PATH_JOB_TEMPLATE_REGEX, "{id}"), http.HandlerFunc(api.handlerJobTemplate)).Methods(http.MethodGet)
	// api_route.Handle("/goals", http.HandlerFunc(api.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	api_route.Handle(fmt.Sprintf(API_PATH_GOALS_NODE_REGEX, "{nodeName}"), http.HandlerFunc(api.handlerGoalForNode)).Methods(http.MethodGet)
	if api.enablePushNotification {
		logger.Info.Printf("Enabling push notification. Nodes can connect to /goals/{nodeName}/stream to get notification from the cloud scheduler.")
		api_route.Handle(fmt.Sprintf(API_PATH_GOALS_NODE_STREAM_REGEX, "{nodeName}"), http.HandlerFunc(api.handlerGoalStreamForNode)).Methods(http.MethodGet)
	}
	api.managementRouter = mux.NewRouter()
	mr := api.managementRouter
	management_route := mr.PathPrefix(API_V1_VERSION).Subrouter()
	if prometheusGatherer != nil {
		management_route.Handle(MANAGEMENT_API_PATH_SYSTEM_METRICS,
			promhttp.HandlerFor(prometheusGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true})).
			Methods(http.MethodGet)
	}
	management_route.Handle(MANAGEMENT_API_PATH_DB, http.HandlerFunc(api.handleDB)).
		Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	management_route.Handle(MANAGEMENT_API_PATH_DATA_JOBS, http.HandlerFunc(api.handleDataJobs)).
		Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	management_route.Handle(MANAGEMENT_API_PATH_DATA_PLUGINS, http.HandlerFunc(api.handleDataPlugins)).
		Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	management_route.Handle(MANAGEMENT_API_PATH_DATA_PLUGINS_WHITELIST, http.HandlerFunc(api.handleDataPluginsWhitelist)).
		Methods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
	management_route.Handle(MANAGEMENT_API_PATH_DATA_NODES, http.HandlerFunc(api.handleDataNodes)).
		Methods(http.MethodGet, http.MethodPost, http.MethodPut)

}

func (api *APIServer) Run() {
	APIAddressPort := fmt.Sprintf("0.0.0.0:%d", api.port)
	logger.Info.Printf("API server starts at %q...", APIAddressPort)
	managementAddressPort := fmt.Sprintf("0.0.0.0:%d", api.managementPort)
	logger.Info.Printf("Management server starts at %q...", managementAddressPort)

	// Added as requested for browser support
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	credentialOK := handlers.AllowCredentials()
	cors := handlers.CORS(headersOk, originsOk, methodsOk, credentialOK)(api.apiRouter)
	go http.ListenAndServe(managementAddressPort, handlers.LoggingHandler(os.Stdout, api.managementRouter))
	logger.Info.Fatalln(http.ListenAndServe(APIAddressPort, handlers.LoggingHandler(os.Stdout, cors)))
}

func (api *APIServer) handlerCreateJob(w http.ResponseWriter, r *http.Request) {
	user, err := api.authenticate(r)
	if err != nil {
		response := datatype.NewAPIMessageBuilder()
		response.AddError(err.Error())
		respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
		return
	}
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
			User: user.GetUserName(),
		}
	case http.MethodPost:
		defer r.Body.Close()
		// The query includes a full job description
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		} else {
			err = yaml.Unmarshal(blob, &newJob)
			// Make sure job owner is the same with the authenticated user
			newJob.User = user.GetUserName()
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
		AddEntity("state", datatype.JobCreated).
		Build()
	respondJSON(w, http.StatusOK, response.ToJson())
}

func (api *APIServer) handlerEditJob(w http.ResponseWriter, r *http.Request) {
	user, err := api.authenticate(r)
	if err != nil {
		response := datatype.NewAPIMessageBuilder()
		response.AddError(err.Error())
		respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
		return
	}
	queries := r.URL.Query()
	if _, exist := queries["id"]; exist {
		jobID := queries.Get("id")
		oldJob, err := api.cloudScheduler.GoalManager.GetJob(jobID)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
		defer r.Body.Close()
		// TODO: the API always assumes that the body contains job content
		updatedJob := datatype.NewJob("", "", "")
		// The query includes a full job description
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
		err = yaml.Unmarshal(blob, &updatedJob)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
		updatedJob.JobID = jobID
		if oldJob.User != user.GetUserName() {
			logger.Info.Printf("user %q does not own the job %s", user.GetUserName(), jobID)
			if queries.Get("override") == "true" {
				logger.Info.Printf("user %q is attempting to override job %q owned by %s", user.GetUserName(), jobID, oldJob.User)
				if user.Auth.IsSuperUser {
					logger.Info.Printf("user %q is a super user. overriding permitted", user.GetUserName())
					// keep the original user name
					updatedJob.User = oldJob.User
				} else {
					response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("User %s does not have permission to override to the job", user.GetUserName())).Build()
					respondJSON(w, http.StatusBadRequest, response.ToJson())
					return
				}
			} else {
				response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("User %s does not have access to the job", user.GetUserName())).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
		} else {
			updatedJob.User = user.GetUserName()
		}
		// Remove science goal of old Job if exists
		if oldJob.ScienceGoal != nil {
			api.cloudScheduler.GoalManager.RemoveScienceGoal(oldJob.ScienceGoal.ID)
		}
		updatedJob.Drafted()
		api.cloudScheduler.GoalManager.UpdateJob(updatedJob, false)
		response := datatype.NewAPIMessageBuilder().AddEntity("job_id", jobID).AddEntity("state", datatype.JobDrafted)
		respondJSON(w, http.StatusOK, response.Build().ToJson())
		return
	} else {
		response := datatype.NewAPIMessageBuilder().AddError("job_id is required").Build()
		respondJSON(w, http.StatusBadRequest, response.ToJson())
		return
	}
}

func (api *APIServer) handlerSubmitJobs(w http.ResponseWriter, r *http.Request) {
	user, err := api.authenticate(r)
	if err != nil {
		response := datatype.NewAPIMessageBuilder()
		response.AddError(err.Error())
		respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
		return
	}
	// Update user permission table for validating user against node access permission
	err = api.authenticator.UpdatePermissionTableForUser(user)
	if err != nil {
		response := datatype.NewAPIMessageBuilder()
		response.AddError(err.Error())
		respondJSON(w, http.StatusInternalServerError, response.Build().ToJson())
		return
	}
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
		if jobID := queries.Get("id"); jobID != "" {
			existingJob, err := api.cloudScheduler.GoalManager.GetJob(jobID)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
			if existingJob.User != user.GetUserName() {
				logger.Info.Printf("user %q does not own the job %s", user.GetUserName(), jobID)
				if queries.Get("override") == "true" {
					logger.Info.Printf("user %q is attempting to override job %q owned by %s", user.GetUserName(), jobID, existingJob.User)
					if user.Auth.IsSuperUser {
						logger.Info.Printf("user %q is a super user. overriding permitted", user.GetUserName())
					} else {
						response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("User %s does not have permission to override to the job", user.GetUserName())).Build()
						respondJSON(w, http.StatusBadRequest, response.ToJson())
						return
					}
				} else {
					response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("User %s does not have access to the job", user.GetUserName())).Build()
					respondJSON(w, http.StatusBadRequest, response.ToJson())
					return
				}
			}
			// TODO: we should not commit to change on the existing goal of job when --dry-run is given
			errorList := api.cloudScheduler.ValidateJobAndCreateScienceGoalForExistingJob(queries.Get("id"), user, flagDryRun)
			if len(errorList) > 0 {
				response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("%v", errorList)).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			} else {
				response := datatype.NewAPIMessageBuilder().AddEntity("job_id", queries.Get("id"))
				if flagDryRun {
					response = response.AddEntity("dryrun", true)
				} else {
					response = response.AddEntity("state", datatype.JobSubmitted)
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
		defer r.Body.Close()
		// TODO: We will need to add an error hanlding when people request this without a body
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
			newJob.User = user.GetUserName()
			sg, errorList := api.cloudScheduler.ValidateJobAndCreateScienceGoal(newJob, user)
			if len(errorList) > 0 {
				response := datatype.NewAPIMessageBuilder().
					AddEntity("job_name", newJob.Name).
					AddEntity("message", "validation failed. Please revise the job and try again.").
					AddError(fmt.Sprintf("%v", errorList)).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			} else {
				newJob.ScienceGoal = sg
				response := datatype.NewAPIMessageBuilder().AddEntity("job_name", newJob.Name)
				if flagDryRun {
					response = response.AddEntity("dryrun", true)
				} else {
					jobID := api.cloudScheduler.GoalManager.AddJob(newJob)
					newJob.UpdateJobID(jobID)
					api.cloudScheduler.GoalManager.UpdateJob(newJob, true)
					response = response.AddEntity("job_id", jobID).
						AddEntity("state", datatype.JobSubmitted)
				}
				respondJSON(w, http.StatusOK, response.Build().ToJson())
				return
			}
		}
	}
}

// handlerJobs handles getting jobs requests. It returns the full list of current jobs
// if user token is not provided. If provided, it returns only the list owned by the token owner.
func (api *APIServer) handlerJobs(w http.ResponseWriter, r *http.Request) {
	userName := ""
	if hasToken(r) {
		user, err := api.authenticate(r)
		if err != nil {
			response := datatype.NewAPIMessageBuilder()
			response.AddError(err.Error())
			respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
			return
		}
		userName = user.GetUserName()
	}
	if r.Method == http.MethodGet {
		response := datatype.NewAPIMessageBuilder()
		var jobs []*datatype.Job
		jobs = api.cloudScheduler.GoalManager.GetJobs(userName)
		for _, job := range jobs {
			response.AddEntity(job.JobID, job)
		}
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	}
}

// handlerJobStatus returns details of jobs.
func (api *APIServer) handlerJobStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Since handlerJobs is open to public, we do not need to check authentication here.
	//       However, we may want to revisit this if this function returns more than what handlerJobs returns
	// user, err := api.authenticate(r)
	// if err != nil {
	// 	response := datatype.NewAPIMessageBuilder()
	// 	response.AddError(err.Error())
	// 	respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
	// 	return
	// }
	vars := mux.Vars(r)
	if r.Method == http.MethodGet {
		response := datatype.NewAPIMessageBuilder()
		job, err := api.cloudScheduler.GoalManager.GetJob(vars["id"])
		if err != nil {
			response.AddError(err.Error())
			respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
			return
		}
		// if user.Auth.IsSuperUser {
		// 	response.AddEntity(vars["id"], job)
		// 	respondJSON(w, http.StatusOK, response.Build().ToJson())
		// 	return
		// }
		// if job.User != user.GetUserName() {
		// 	response.AddError(fmt.Sprintf("User %s does not have permission to view the job %s", user.GetUserName(), vars["id"]))
		// 	respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
		// 	return
		// }
		// response.AddEntity(vars["id"], job)
		blob, err := httpSensitiveJsonMarshal(job)
		if err != nil {
			response.AddError(err.Error())
			respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
			return
		}
		respondJSON(w, http.StatusOK, blob)
		// respondYAML(w, http.StatusOK, job)
	}
}

func (api *APIServer) handlerJobRemove(w http.ResponseWriter, r *http.Request) {
	response := datatype.NewAPIMessageBuilder()
	user, err := api.authenticate(r)
	if err != nil {
		response.AddError(err.Error())
		respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
		return
	}
	vars := mux.Vars(r)
	queries := r.URL.Query()
	jobID := vars["id"]
	job, err := api.cloudScheduler.GoalManager.GetJob(jobID)
	if err != nil {
		response.AddError(err.Error())
		respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
		return
	}
	if job.User != user.GetUserName() {
		logger.Info.Printf("user %q does not own the job %s", user.GetUserName(), jobID)
		if queries.Get("override") == "true" {
			logger.Info.Printf("user %q is attempting to override job %q owned by %s", user.GetUserName(), jobID, job.User)
			if user.Auth.IsSuperUser {
				logger.Info.Printf("user %q is a super user. overriding permitted", user.GetUserName())
			} else {
				response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("User %s does not have permission to override to the job", user.GetUserName())).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
		} else {
			response := datatype.NewAPIMessageBuilder().AddError(fmt.Sprintf("user %q is not the owner of job %q", user.GetUserName(), jobID)).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		}
	}
	// Suspend the job instead of removal if suspend flag is given
	suspend := queries.Get("suspend")
	if suspend == "true" {
		err := api.cloudScheduler.GoalManager.SuspendJob(jobID)
		if err != nil {
			response.AddEntity("job_id", jobID).
				AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
			return
		}
		response.AddEntity("job_id", jobID).
			AddEntity("state", datatype.JobSuspended)
		respondJSON(w, http.StatusOK, response.Build().ToJson())
		return
	}
	force := false
	if _, exist := queries["force"]; exist {
		forceString := queries.Get("force")
		if forceString == "true" {
			force = true
		}
	}
	err = api.cloudScheduler.GoalManager.RemoveJob(jobID, force)
	if err != nil {
		response.AddEntity("job_id", jobID).
			AddError(err.Error()).Build()
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	} else {
		response.AddEntity("job_id", jobID).
			AddEntity("state", datatype.JobRemoved)
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	}
}

func (api *APIServer) handlerJobTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if r.Method == http.MethodGet {
		response := datatype.NewAPIMessageBuilder()
		job, err := api.cloudScheduler.GoalManager.GetJob(vars["id"])
		if err != nil {
			response.AddError(err.Error())
			respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
			return
		}
		blob, err := httpSensitiveYamlMarshal(job.ConvertToTemplate())
		if err != nil {
			response.AddError(err.Error())
			respondJSON(w, http.StatusBadRequest, response.Build().ToJson())
			return
		}
		respondJSON(w, http.StatusOK, blob)
		// respondYAML(w, http.StatusOK, job.ConvertToTemplate())
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
	blob, err := httpSensitiveJsonMarshal(goals)
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
	// To prevent nginx proxy from keeping buffer of data
	w.Header().Set("X-Accel-Buffering", "no")
	c := make(chan datatype.Event, 1)
	api.subscribe(nodeName, c)
	defer api.unsubscribe(nodeName, c)
	var goals []*datatype.ScienceGoal
	for _, g := range api.cloudScheduler.GoalManager.GetScienceGoalsForNode(nodeName) {
		goals = append(goals, g.ShowMyScienceGoal(nodeName))
	}
	// if no science goal is assigned to the node return an empty list []
	// returning null may raise an exception in edge scheduler
	if len(goals) < 1 {
		event := datatype.NewSchedulerEventBuilder(datatype.EventGoalStatusUpdated).
			AddEntry("goals", "[]").
			Build().(datatype.SchedulerEvent)
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.ToString(), event.GetEntry("goals")); err != nil {
			return
		}
		flusher.Flush()
	} else {
		blob, err := httpSensitiveJsonMarshal(goals)
		if err != nil {
			logger.Error.Printf("Failed to compress goals for node %q before pushing", nodeName)
		} else {
			event := datatype.NewSchedulerEventBuilder(datatype.EventGoalStatusUpdated).
				AddEntry("goals", string(blob)).
				Build().(datatype.SchedulerEvent)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.ToString(), event.GetEntry("goals")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
	for {
		select {
		case event := <-c:
			e := event.(datatype.SchedulerEvent)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.ToString(), e.GetEntry("goals")); err != nil {
				return
			}
			// option 1. send a small heartbeat to the client to keep the connection and notify if it fails
			// option 2. check if the connection is closed so that it can clean up
			flusher.Flush()
		case <-r.Context().Done():
			flusher.Flush()
			return
		}
	}
}

func (api *APIServer) handleDB(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// dump the current DB
		err := api.cloudScheduler.GoalManager.DumpDB(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost, http.MethodPut:
		// TODO: To be implemented. This will
		//       1. overwrite the db file specified in the config with given byte blob
		//       2. terminate the schedueler to safely load the overwritten db file
		http.Error(w, "Not implemented", http.StatusInternalServerError)
	default:
		http.Error(w, fmt.Sprintf("the HTTP method %q is not supported", r.Method), http.StatusBadRequest)
	}
}

func (api *APIServer) handleDataJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		jobID := queries.Get("id")
		if jobID == "" {
			http.Error(w, "No id is specified", http.StatusBadRequest)
			return
		}
		blob, err := api.cloudScheduler.GoalManager.GetRecord(jobID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, http.StatusOK, blob)
	case http.MethodPost:
		queries := r.URL.Query()
		jobID := queries.Get("id")
		if jobID == "" {
			http.Error(w, "No id is specified", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = api.cloudScheduler.GoalManager.SetRecord(jobID, blob)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		http.Error(w, "Not implemented", http.StatusInternalServerError)
	default:
		http.Error(w, fmt.Sprintf("the HTTP method %q is not supported", r.Method), http.StatusBadRequest)

	}
}

func (api *APIServer) handleDataPlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		flagReload := queries.Get("reload")
		if flagReload == "true" {
			if err := api.cloudScheduler.Validator.LoadDatabase(); err != nil {
				http.Error(w, fmt.Sprintf("error on loading database: %s", err.Error()), http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		} else {
			response := datatype.NewAPIMessageBuilder()
			pluginImages := []string{}
			for image := range api.cloudScheduler.Validator.Plugins {
				pluginImages = append(pluginImages, image)
			}
			response.AddEntity("plugins", pluginImages)
			respondJSON(w, http.StatusOK, response.Build().ToJson())
		}
	case http.MethodPost:
		fallthrough
	case http.MethodPut:
		http.Error(w, "Not implemented", http.StatusInternalServerError)
	default:
		http.Error(w, fmt.Sprintf("the HTTP method %q is not supported", r.Method), http.StatusBadRequest)
	}
}

func (api *APIServer) handleDataPluginsWhitelist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		flagReload := queries.Get("reload")
		if flagReload == "true" {
			api.cloudScheduler.Validator.LoadPluginWhitelist()
		}
		response := datatype.NewAPIMessageBuilder()
		response.AddEntity("whitelist", api.cloudScheduler.Validator.ListPluginWhitelist())
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	case http.MethodPost:
		defer r.Body.Close()
		if blob, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			response := datatype.NewAPIMessageBuilder()
			logger.Info.Printf("adding plugin whitelist %q", blob)
			api.cloudScheduler.Validator.AddPluginWhitelist(string(blob))
			api.cloudScheduler.Validator.WritePluginWhitelist()
			response.AddEntity("whitelist", string(blob))
			response.AddEntity("status", "success")
			respondJSON(w, http.StatusOK, response.Build().ToJson())
		}
	case http.MethodPut:
		http.Error(w, "Not implemented", http.StatusInternalServerError)
	case http.MethodDelete:
		defer r.Body.Close()
		if blob, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			response := datatype.NewAPIMessageBuilder()
			logger.Info.Printf("removing plugin whitelist %q", blob)
			api.cloudScheduler.Validator.RemovePluginWhitelist(string(blob))
			api.cloudScheduler.Validator.WritePluginWhitelist()
			response.AddEntity("whitelist", string(blob))
			response.AddEntity("status", "success")
			respondJSON(w, http.StatusOK, response.Build().ToJson())
		}
	default:
		http.Error(w, fmt.Sprintf("the HTTP method %q is not supported", r.Method), http.StatusBadRequest)
	}
}

func (api *APIServer) handleDataNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		flagReload := queries.Get("reload")
		if flagReload == "true" {
			if err := api.cloudScheduler.Validator.LoadDatabase(); err != nil {
				http.Error(w, fmt.Sprintf("error on loading database: %s", err.Error()), http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		} else {
			http.Error(w, "the reload query must be reload=true", http.StatusBadRequest)
		}
	case http.MethodPost:
		fallthrough
	case http.MethodPut:
		http.Error(w, "Not implemented", http.StatusInternalServerError)
	default:
		http.Error(w, fmt.Sprintf("the HTTP method %q is not supported", r.Method), http.StatusBadRequest)
	}
}

func (api *APIServer) authenticate(r *http.Request) (*User, error) {
	token, err := extractToken(r)
	if err != nil {
		return nil, err
	}
	user, err := api.authenticator.Authenticate(token)
	if err != nil {
		return nil, fmt.Errorf("Authentication failed: %s", err.Error())
	}
	// if user.Auth.Active == false {
	// 	return user, fmt.Errorf("User %q is inactive. Please contact administrator.", user.GetUserName())
	// }
	return user, nil
}

func respondJSON(w http.ResponseWriter, statusCode int, data []byte) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(statusCode)

	// json.NewEncoder(w).Encode(data)
	s, err := yaml.Marshal(data)
	if err == nil {
		w.Write(s)
	}
}

func hasToken(r *http.Request) bool {
	if auth := r.Header.Get("Authorization"); auth == "" {
		return false
	} else {
		return true
	}
}

func extractToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("No token found")
	}
	// TODO: This prefix may be Beehive/project dependent
	const prefix = "Sage "
	if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
		return "", fmt.Errorf("Token is not a Sage token")
	}
	token := auth[len(prefix):]
	if len(token) == 0 {
		return "", fmt.Errorf("Token not found")
	}
	return token, nil
}

func httpSensitiveJsonMarshal(o interface{}) ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(bf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", " ")
	err := encoder.Encode(o)
	return bf.Bytes(), err
}

func httpSensitiveYamlMarshal(o interface{}) ([]byte, error) {
	return yaml.Marshal(o)
	// bf := bytes.NewBuffer([]byte{})
	// encoder := yaml.NewEncoder(bf)
	// err := encoder.Encode(o)
	// return bf.Bytes(), err
}
