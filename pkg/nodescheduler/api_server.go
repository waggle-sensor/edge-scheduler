package nodescheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	// "net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	// "github.com/urfave/negroni"
)

type APIServer struct {
	version       string
	port          int
	mainRouter    *mux.Router
	nodeScheduler *NodeScheduler
}

func (api *APIServer) Run() {
	api_address_port := fmt.Sprintf("0.0.0.0:%d", api.port)
	logger.Info.Printf("API server starts at %q...", api_address_port)
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Node Scheduler (`+api.nodeScheduler.NodeID+`)", "version":"`+api.version+`"}`)
	})
	api_route := r.PathPrefix("/api/v1").Subrouter()
	// mux := http.NewServeMux()
	// mux.HandleFunc("/debug/pprof", pprof.Index)
	// go http.ListenAndServe(":18080", nil)
	api_route.Handle("/goals", http.HandlerFunc(api.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	api_route.Handle("/schedule", http.HandlerFunc(api.handlerSchedule)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	// api_route.Handle("/status/queue/waiting", http.HandlerFunc(api.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	logger.Info.Fatalln(http.ListenAndServe(api_address_port, r))
}

func respondJSON(w http.ResponseWriter, statusCode int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}

func (api *APIServer) handlerGoals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// clauses := PrintClauses()
		// respondJSON(w, http.StatusOK, clauses)
	case http.MethodPost:
		var newGoals []datatype.ScienceGoal
		defer r.Body.Close()
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		} else {
			logger.Debug.Printf("%s", string(blob))
			err = json.Unmarshal(blob, &newGoals)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
		}
		logger.Info.Printf("Adding goals by the REST call.")
		for _, goal := range newGoals {
			api.nodeScheduler.GoalManager.AddGoal(&goal)
		}
		response := datatype.NewAPIMessageBuilder().AddEntity("status", "success").Build()
		respondJSON(w, http.StatusOK, response.ToJson())
	}
}

func (api *APIServer) handlerSchedule(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// clauses := PrintClauses()
		// respondJSON(w, http.StatusOK, clauses)
	case http.MethodPost:
		var newPlugin datatype.Plugin
		defer r.Body.Close()
		blob, err := io.ReadAll(r.Body)
		if err != nil {
			response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
			respondJSON(w, http.StatusBadRequest, response.ToJson())
			return
		} else {
			logger.Debug.Printf("%s", string(blob))
			err = json.Unmarshal(blob, &newPlugin)
			if err != nil {
				response := datatype.NewAPIMessageBuilder().AddError(err.Error()).Build()
				respondJSON(w, http.StatusBadRequest, response.ToJson())
				return
			}
		}
		logger.Info.Printf("locally requested to add plugin %q to schedule", newPlugin.Name)
		e := datatype.NewSchedulerEventBuilder(datatype.EventPluginStatusQueued).AddReason("locally submitted").Build()
		// api.nodeScheduler.LogToBeehive.SendWaggleMessageOnNodeAsync(response.ToWaggleMessage(), "node")
		pr := datatype.NewPluginRuntimeWithScienceRule(newPlugin, datatype.ScienceRule{})
		pr.EnablePluginController = true
		api.nodeScheduler.readyQueue.Push(pr)
		api.nodeScheduler.chanNeedScheduling <- e

		response := datatype.NewAPIMessageBuilder().
			AddEntity("plugin_name", pr.Plugin.Name).
			AddEntity("status", "success").Build()
		respondJSON(w, http.StatusOK, response.ToJson())
	}
}
