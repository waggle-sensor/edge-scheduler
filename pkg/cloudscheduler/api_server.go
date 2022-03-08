package cloudscheduler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
	// "github.com/urfave/negroni"
)

type APIServer struct {
	version        string
	port           int
	mainRouter     *mux.Router
	cloudScheduler *CloudScheduler
}

func (api *APIServer) Run() {
	api_address_port := fmt.Sprintf("0.0.0.0:%d", api.port)
	logger.Info.Printf("API server starts at %q...", api_address_port)
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Cloud Scheduler (`+api.cloudScheduler.Name+`)", "version":"`+api.version+`"}`)
	})
	api_route := r.PathPrefix("/api/v1").Subrouter()
	api_route.Handle("/create", http.HandlerFunc(api.handlerCreateJob)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle("/edit", http.HandlerFunc(api.handlerEditJob)).Methods(http.MethodGet)
	api_route.Handle("/submit", http.HandlerFunc(api.handlerSubmitJobs)).Methods(http.MethodPost, http.MethodPut)
	api_route.Handle("/jobs", http.HandlerFunc(api.handlerJobs)).Methods(http.MethodGet)
	api_route.Handle("/jobs/{name}/status", http.HandlerFunc(api.handlerJobStatus)).Methods(http.MethodGet)
	// api.Handle("/goals", http.HandlerFunc(cs.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	// api.Handle("/goals/{nodeName}", http.HandlerFunc(cs.handlerGoalForNode)).Methods(http.MethodGet
	logger.Info.Fatalln(http.ListenAndServe(api_address_port, api.mainRouter))
}

func (api *APIServer) handlerCreateJob(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		if _, exist := queries["name"]; !exist {
			response := datatype.NewAPIMessageBuilder().AddError("name field is required").Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
		} else {
			name := queries.Get("name")
			newJob := &datatype.Job{
				Name: name,
			}
			api.cloudScheduler.GoalManager.AddJob(newJob)
			response := datatype.NewAPIMessageBuilder().AddEntity("job_name", name).Build()
			respondJSON(w, http.StatusOK, response.ToJson())
		}
	case http.MethodPost:
		// The query includes a full job description
	}
}

func (api *APIServer) handlerEditJob(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	if _, exist := queries["name"]; !exist {
		respondJSON(w, http.StatusBadRequest, []byte{})
	} else {
		respondJSON(w, http.StatusOK, []byte{})
	}
}

func (api *APIServer) handlerSubmitJobs(w http.ResponseWriter, r *http.Request) {
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
	respondJSON(w, http.StatusOK, []byte{})
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
		for jobName, job := range jobs {
			response.AddEntity(jobName, job)
		}
		respondJSON(w, http.StatusOK, response.Build().ToJson())
	}
}

func (api *APIServer) handlerJobStatus(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	if r.Method == http.MethodGet {
		log.Printf("hit GET")
		// logger.Info.Printf("Job status of %s", vars["id"])
		// if goal, err := cs.GoalManager.GetScienceGoal(vars["id"]); err == nil {
		// 	respondJSON(w, http.StatusOK, goal)
		// } else {
		// 	respondJSON(w, http.StatusOK, "")
		// }
		api.cloudScheduler.GoalManager.GetJobs()
		respondJSON(w, http.StatusOK, []byte{})
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
	// vars := mux.Vars(r)
	if r.Method == http.MethodGet {
		// nodeName := vars["nodeName"]
		// goals := cs.GoalManager.GetScienceGoalsForNode(nodeName)
		// dat, _ := yaml.Marshal(goals)
		// respondYAML(w, http.StatusOK, goals)
		// respondYAML(w, http.StatusOK, `[{"response": "No goals found"}]`)
		respondJSON(w, http.StatusOK, []byte{})
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
