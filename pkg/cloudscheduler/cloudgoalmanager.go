package cloudscheduler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

const jobBucketName = "jobs"

// CloudGoalManager structs a goal manager for cloudscheduler
type CloudGoalManager struct {
	scienceGoals map[string]*datatype.ScienceGoal
	Notifier     *interfacing.Notifier
	mu           sync.Mutex
	dataPath     string
	jobDB        *bolt.DB
}

func (cgm *CloudGoalManager) AddJob(job *datatype.Job) string {
	job.Created()
	cgm.jobDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		jobID, _ := b.NextSequence()
		job.JobID = fmt.Sprintf("%d", int(jobID))
		buf, err := json.Marshal(job)
		if err != nil {
			return err
		}
		b.Put([]byte(job.JobID), []byte(buf))
		return nil
	})
	return job.JobID
}

func (cgm *CloudGoalManager) GetJobs(userName string) (jobs []*datatype.Job) {
	err := cgm.jobDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		return b.ForEach(func(k, v []byte) error {
			var j datatype.Job
			// TODO: This needs to return the error to the caller. It does not for now.
			if err := json.Unmarshal(v, &j); err != nil {
				return err
			}
			// If username is given we return jobs that belong to the user
			if userName != "" {
				if j.User == userName {
					jobs = append(jobs, &j)
				}
			} else {
				jobs = append(jobs, &j)
			}
			return nil
		})
	})
	if err != nil {
		logger.Error.Printf("Error from DB: %s", err.Error())
	}
	return
}

func (cgm *CloudGoalManager) GetJob(jobID string) (job *datatype.Job, err error) {
	err = cgm.jobDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		v := b.Get([]byte(jobID))
		if v == nil {
			return fmt.Errorf("Job ID %q does not exist", jobID)
		}
		var j datatype.Job
		if err := json.Unmarshal(v, &j); err != nil {
			return err
		}
		job = &j
		return nil
	})
	return
}

// GetRecord returns a job as a byte blob. This function should only be used for management purpose.
func (cgm *CloudGoalManager) GetRecord(jobID string) (blob []byte, err error) {
	err = cgm.jobDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		blob = b.Get([]byte(jobID))
		if len(blob) < 1 {
			return fmt.Errorf("Job ID %q does not exist", jobID)
		}
		return nil
	})
	return
}

func (cgm *CloudGoalManager) SetRecord(jobID string, blob []byte) error {
	return cgm.jobDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		b.Put([]byte(jobID), []byte(blob))
		return nil
	})
}

func (cgm *CloudGoalManager) UpdateJob(job *datatype.Job, submit bool) (err error) {
	// update the status before puting the job to the database
	if submit {
		job.Submitted()
	}
	err = cgm.jobDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		buf, err := json.Marshal(job)
		if err != nil {
			return err
		}
		b.Put([]byte(job.JobID), []byte(buf))
		return nil
	})
	if err != nil {
		return
	}
	// send an event for scheduling the science goal
	if submit {
		newScienceGoal := job.ScienceGoal
		cgm.UpdateScienceGoal(newScienceGoal)
		event := datatype.NewEventBuilder(datatype.EventGoalStatusSubmitted).AddGoal(newScienceGoal).Build()
		cgm.Notifier.Notify(event)
	}
	return
}

func (cgm *CloudGoalManager) SuspendJob(jobID string) (err error) {
	var job datatype.Job
	err = cgm.jobDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		v := b.Get([]byte(jobID))
		if v == nil {
			return fmt.Errorf("Job ID %q does not exist", jobID)
		}
		if err := json.Unmarshal(v, &job); err != nil {
			return err
		}
		job.Suspended()
		buf, err := json.Marshal(job)
		if err != nil {
			return err
		}
		return b.Put([]byte(job.JobID), []byte(buf))
	})
	if err != nil {
		return
	}
	event := datatype.NewEventBuilder(datatype.EventJobStatusSuspended).
		AddJob(&job).
		AddReason("Suspended by user").Build()
	cgm.Notifier.Notify(event)
	return
}

func (cgm *CloudGoalManager) RemoveJob(jobID string, force bool) (err error) {
	var job datatype.Job
	err = cgm.jobDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		v := b.Get([]byte(jobID))
		if v == nil {
			return fmt.Errorf("Job ID %q does not exist", jobID)
		}
		if err := json.Unmarshal(v, &job); err != nil {
			return err
		}
		if job.State.GetState() == datatype.JobRunning && !force {
			return fmt.Errorf("Failed to remove job %q as it is in running state. Suspend it first or specify force=true", jobID)
		}
		job.Removed()
		buf, err := json.Marshal(job)
		if err != nil {
			return err
		}
		return b.Put([]byte(job.JobID), []byte(buf))
	})
	if err != nil {
		return
	}
	event := datatype.NewEventBuilder(datatype.EventJobStatusRemoved).
		AddJob(&job).
		Build()
	cgm.Notifier.Notify(event)
	return
}

func (cgm *CloudGoalManager) RemoveScienceGoal(goalID string) error {
	cgm.mu.Lock()
	defer cgm.mu.Unlock()
	if goal, exist := cgm.scienceGoals[goalID]; exist {
		delete(cgm.scienceGoals, goal.ID)
		return nil
	} else {
		return fmt.Errorf("Failed to find science goal %q to remove", goalID)
	}
}

// UpdateScienceGoal stores given science goal
func (cgm *CloudGoalManager) UpdateScienceGoal(scienceGoal *datatype.ScienceGoal) error {
	cgm.mu.Lock()
	defer cgm.mu.Unlock()
	cgm.scienceGoals[scienceGoal.ID] = scienceGoal

	// Send the updated science goal to all subject edge schedulers
	// if cgm.rmqHandler != nil {
	// 	// TODO: Refine what to send to RMQ for edge scheduler
	// 	// Send the updates
	// 	for _, subGoal := range scienceGoal.SubGoals {
	// 		message, err := yaml.Marshal([]*datatype.ScienceGoal{scienceGoal})
	// 		if err != nil {
	// 			logger.Error.Printf("Unable to parse the science goal <%s> into YAML: %s", scienceGoal.ID, err.Error())
	// 			continue
	// 		}
	// 		logger.Debug.Printf("%+v", string(message))
	// 		cgm.rmqHandler.SendYAML(subGoal.Name, message)
	// 	}
	// }

	return nil
}

// GetScienceGoal returns the science goal matching to given science goal ID
func (cgm *CloudGoalManager) GetScienceGoal(goalID string) (*datatype.ScienceGoal, error) {
	cgm.mu.Lock()
	defer cgm.mu.Unlock()
	if goal, exist := cgm.scienceGoals[goalID]; exist {
		return goal, nil
	}
	return nil, fmt.Errorf("Failed to find goal %q", goalID)
}

// GetScienceGoalsForNode returns a list of goals associated to given node.
func (cgm *CloudGoalManager) GetScienceGoalsForNode(nodeName string) (goals []*datatype.ScienceGoal) {
	for _, scienceGoal := range cgm.scienceGoals {
		for _, subGoal := range scienceGoal.SubGoals {
			if strings.ToLower(subGoal.Name) == strings.ToLower(nodeName) {
				goals = append(goals, scienceGoal)
			}
		}
	}
	return
}

func (cgm *CloudGoalManager) EditRecord(job *datatype.Job) error {
	cgm.mu.Lock()
	defer cgm.mu.Unlock()
	return cgm.jobDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		buf, err := json.Marshal(job)
		if err != nil {
			return err
		}
		b.Put([]byte(job.JobID), []byte(buf))
		return nil
	})
}

func (cgm *CloudGoalManager) OpenJobDB() error {
	db, err := bolt.Open(path.Join(cgm.dataPath, "job.db"), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	cgm.jobDB = db
	cgm.jobDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(jobBucketName))
		return err
	})
	return nil
}

func (cgm *CloudGoalManager) DumpDB(w http.ResponseWriter) error {
	return cgm.jobDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="my.db"`)
		w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
		_, err := tx.WriteTo(w)
		return err
	})
}

func (cgm *CloudGoalManager) LoadScienceGoalsFromJobDB() error {
	cgm.jobDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(jobBucketName))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist", jobBucketName)
		}
		return b.ForEach(func(k, v []byte) error {
			var j datatype.Job
			if err := json.Unmarshal(v, &j); err != nil {
				return err
			}
			switch j.State.GetState() {
			case datatype.JobSubmitted, datatype.JobRunning:
				cgm.UpdateScienceGoal(j.ScienceGoal)
			}
			return nil
		})
	})
	return nil
}
