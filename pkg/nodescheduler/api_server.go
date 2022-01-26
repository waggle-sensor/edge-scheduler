package nodescheduler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
	// "github.com/urfave/negroni"
)

const (
	GET  = "GET"
	POST = "POST"
	PUT  = "PUT"
)

var (
	// Channels for IPC
	chanFromMeasure = make(chan RMQMessage)
)

type APIServer struct {
	mainRouter    *mux.Router
	nodeScheduler *NodeScheduler
}

func NewAPIServer() *APIServer {
	return &APIServer{}
}

func (api *APIServer) Run() {
	log.Printf("initializing...")
	api.mainRouter = mux.NewRouter()
	r := api.mainRouter
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Node scheduler", "version": "0.0.5"}`)
	})
	api_route := r.PathPrefix("/api/v1").Subrouter()
	api_route.Handle("/kb/rules", http.HandlerFunc(handlerClauses)).Methods(http.MethodGet, http.MethodPost)
	api_route.Handle("/kb/senses", http.HandlerFunc(api.handlerSenses)).Methods(http.MethodGet, http.MethodPost, http.MethodDelete)
	api_route.Handle("/goals", http.HandlerFunc(api.handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	logger.Info.Fatalln(http.ListenAndServe("0.0.0.0:8080", r))
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

func handlerClauses(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		clauses := ""
		respondJSON(w, http.StatusOK, clauses)
	} else if r.Method == POST {
		r.ParseForm()
		clause := r.Form.Get("clause")
		log.Printf(clause)

		respondJSON(w, http.StatusOK, clause)
	}
}

func (api *APIServer) handlerSenses(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		memory := ""
		respondJSON(w, http.StatusOK, memory)
	} else if r.Method == POST {
		r.ParseForm()
		subject := r.Form.Get("subject")
		value := r.Form.Get("value")
		log.Printf("%s %s", subject, value)
		// Memorize(subject, value)
		// api.nodeScheduler.Knowledgebase.
		respondJSON(w, http.StatusOK, subject+value)
	} else if r.Method == "DELETE" {
		log.Print("hit DELETE")
		r.ParseForm()
		subject := r.Form.Get("subject")
		respondJSON(w, http.StatusOK, subject)
	}
}

func (api *APIServer) handlerGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		// clauses := PrintClauses()
		// respondJSON(w, http.StatusOK, clauses)
	} else if r.Method == POST {
		log.Printf("hit POST")
		respondJSON(w, http.StatusOK, "")
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
		var scienceGoal datatype.ScienceGoal
		_ = yaml.Unmarshal(yamlFile, &scienceGoal)
		logger.Debug.Printf("%v", scienceGoal)
		// RegisterGoal(goal)
		// chanTriggerSchedule <- "received new goal via api"
		// scienceGoal := NewScienceGoal()
		api.nodeScheduler.GoalManager.AddGoal(&scienceGoal)
		respondJSON(w, http.StatusOK, "")
	}
}
