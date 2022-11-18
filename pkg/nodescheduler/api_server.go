package nodescheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	// "github.com/urfave/negroni"
)

var (
	// Channels for IPC
	chanFromMeasure = make(chan RMQMessage)
)

type APIServer struct {
	version       string
	port          int
	mainRouter    *mux.Router
	nodeScheduler *NodeScheduler
}

func NewAPIServer() *APIServer {
	return &APIServer{}
}

func (api *APIServer) Run() {
	api_address_port := "0.0.0.0:8080"
	logger.Info.Printf("API server starts at %q...", api_address_port)
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Node Scheduler (`+api.nodeScheduler.NodeID+`)", "version":"`+api.version+`"}`)
	})
	api_route := r.PathPrefix("/api/v1").Subrouter()
	api_route.Handle("/kb/rules", http.HandlerFunc(api.handlerRules)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle("/kb/senses", http.HandlerFunc(api.handlerSenses)).Methods(http.MethodGet, http.MethodPost, http.MethodDelete)
	api_route.Handle("/goals", http.HandlerFunc(api.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	// api_route.Handle("/status/queue/waiting", http.HandlerFunc(api.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	logger.Info.Fatalln(http.ListenAndServe(api_address_port, r))
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

func (api *APIServer) handlerRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		clauses := ""
		respondJSON(w, http.StatusOK, clauses)
	case http.MethodPost:
		r.ParseForm()
		clause := r.Form.Get("clause")
		log.Printf(clause)
		respondJSON(w, http.StatusOK, clause)
	}
}

func (api *APIServer) handlerSenses(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		queries := r.URL.Query()
		if _, exist := queries["key"]; exist {
			if _, exist := queries["value"]; exist {
				if v, err := strconv.ParseFloat(queries.Get("value"), 64); err != nil {
					api.nodeScheduler.Knowledgebase.AddRawMeasure(queries.Get("key"), queries.Get("value"))
				} else {
					api.nodeScheduler.Knowledgebase.AddRawMeasure(queries.Get("key"), v)
				}
				response := datatype.NewAPIMessageBuilder().AddEntity("status", "success").Build()
				respondJSON(w, http.StatusOK, response.ToJson())
			} else {
				response := datatype.NewAPIMessageBuilder().AddError("no value= found").Build()
				respondJSON(w, http.StatusOK, response.ToJson())
			}
		} else {
			response := datatype.NewAPIMessageBuilder().AddError("no key= found").Build()
			respondJSON(w, http.StatusOK, response.ToJson())
		}
	case http.MethodPost:
		r.ParseForm()
		subject := r.Form.Get("subject")
		value := r.Form.Get("value")
		log.Printf("%s %s", subject, value)
		// Memorize(subject, value)
		// api.nodeScheduler.Knowledgebase.
		respondJSON(w, http.StatusOK, subject+value)
	case http.MethodDelete:
		log.Print("hit DELETE")
		r.ParseForm()
		subject := r.Form.Get("subject")
		respondJSON(w, http.StatusOK, subject)
	}
}

func (api *APIServer) handlerGoals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// clauses := PrintClauses()
		// respondJSON(w, http.StatusOK, clauses)
	case http.MethodPost:
		var newGoals []datatype.ScienceGoal
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
