package nodescheduler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	// "github.com/urfave/negroni"
)

const (
	GET  = "GET"
	POST = "POST"
	PUT  = "PUT"
)

var (
	// Channels for IPC
	chanFromMeasure      chan RMQMessage = make(chan RMQMessage)
	chanTriggerScheduler chan string     = make(chan string)

	mainRouter *mux.Router
)

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
		clauses := PrintClauses()
		respondJSON(w, http.StatusOK, clauses)
	} else if r.Method == POST {
		r.ParseForm()
		clause := r.Form.Get("clause")
		log.Printf(clause)
		AddClause(clause)
		respondJSON(w, http.StatusOK, clause)
	}
}

func handlerSenses(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		memory := PrintMemory()
		respondJSON(w, http.StatusOK, memory)
	} else if r.Method == POST {
		r.ParseForm()
		subject := r.Form.Get("subject")
		value := r.Form.Get("value")
		log.Printf("%s %s", subject, value)
		// Memorize(subject, value)
		respondJSON(w, http.StatusOK, subject+value)
	} else if r.Method == "DELETE" {
		log.Print("hit DELETE")
		r.ParseForm()
		subject := r.Form.Get("subject")
		ClearMemory(subject)
		respondJSON(w, http.StatusOK, subject)
	}
}

func handlerGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		// clauses := PrintClauses()
		// respondJSON(w, http.StatusOK, clauses)
		Test()
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
		var goal Goal
		_ = yaml.Unmarshal(yamlFile, &goal)
		log.Printf("%v", goal)
		RegisterGoal(goal)
		chanTriggerScheduler <- "api server"
		respondJSON(w, http.StatusOK, "")
	}
}

func createRouter() {
	log.Printf("initializing...")
	mainRouter = mux.NewRouter()
	r := mainRouter

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Local scheduler"}`)
	})

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Handle("/kb/rules", http.HandlerFunc(handlerClauses)).Methods(http.MethodGet, http.MethodPost)
	api.Handle("/kb/senses", http.HandlerFunc(handlerSenses)).Methods(http.MethodGet, http.MethodPost, http.MethodDelete)

	api.Handle("/goals", http.HandlerFunc(handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)

	log.Fatalln(http.ListenAndServe("0.0.0.0:8080", r))
}
