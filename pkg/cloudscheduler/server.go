package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"
	// "github.com/urfave/negroni"
)

const (
	GET  = "GET"
	POST = "POST"
	PUT  = "PUT"
)

var (
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

func respondYAML(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(statusCode)

	// json.NewEncoder(w).Encode(data)
	s, err := yaml.Marshal(data)
	if err == nil {
		w.Write(s)
	}
}

func handlerSubmitJobs(w http.ResponseWriter, r *http.Request) {
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
		var job Job
		_ = yaml.Unmarshal(yamlFile, &job)
		job.ID = guuid.New().String()
		log.Printf("%v", job)

		if len(job.PluginTags) > 0 {
			foundPlugins := getPluginsByTags(job.PluginTags)
			job.Plugins = foundPlugins
		}

		if len(job.NodeTags) > 0 {
			foundNodes := getNodesByTags(job.NodeTags)
			job.Nodes = foundNodes
		}

		chanToValidator <- &job
		respondJSON(w, http.StatusOK, "")
	}
}

func handlerJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == GET {
		log.Printf("hit GET")

		respondJSON(w, http.StatusOK, "")
	}
}

func handlerJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if r.Method == GET {
		log.Printf("hit GET")
		InfoLogger.Printf("Job status of %s", vars["id"])
		if goal, ok := scienceGoals[vars["id"]]; ok {
			respondJSON(w, http.StatusOK, goal)
		} else {
			respondJSON(w, http.StatusOK, "")
		}
	}
}

func handlerGoals(w http.ResponseWriter, r *http.Request) {
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

func handlerGoalForNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if r.Method == GET {
		nodeName := vars["nodeName"]
		for _, scienceGoal := range scienceGoals {
			for _, subGoal := range scienceGoal.SubGoals {
				if subGoal.Node.Name == nodeName {
					dat, _ := yaml.Marshal(scienceGoal)
					ioutil.WriteFile("sciencegoal.yaml", dat, 0644)
					respondYAML(w, http.StatusOK, scienceGoal)
					return
				}
			}
		}
		respondYAML(w, http.StatusOK, `{"response": "No goals found"}`)
	}
}

func createRouter() {
	mainRouter = mux.NewRouter()
	r := mainRouter

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"id": "Cloud scheduler"}`)
	})

	api := r.PathPrefix("/api/v1").Subrouter()
	// api.Handle("/kb/rules", http.HandlerFunc(handlerClauses)).Methods(http.MethodGet, http.MethodPost)
	// api.Handle("/kb/senses", http.HandlerFunc(handlerSenses)).Methods(http.MethodGet, http.MethodPost, http.MethodDelete)
	//
	api.Handle("/submit", http.HandlerFunc(handlerSubmitJobs)).Methods(http.MethodPost, http.MethodPut)
	api.Handle("/jobs", http.HandlerFunc(handlerJobs)).Methods(http.MethodGet)
	api.Handle("/jobs/{id}/status", http.HandlerFunc(handlerJobStatus)).Methods(http.MethodGet)
	// api.Handle("/goals", http.HandlerFunc(handlerGoals)).Methods(http.MethodGet, http.MethodPost, http.MethodPut)
	api.Handle("/goals/{nodeName}", http.HandlerFunc(handlerGoalForNode)).Methods(http.MethodGet)

	InfoLogger.Fatalln(http.ListenAndServe("0.0.0.0:9770", r))
}
