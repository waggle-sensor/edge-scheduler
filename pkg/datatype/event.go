package datatype

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sagecontinuum/ses/pkg/logger"
	batchv1 "k8s.io/api/batch/v1"
)

type EventBuilder struct {
	e Event
}

type Event struct {
	Type      EventType
	Timestamp int64
	Meta      map[string]string
}

func NewEventBuilder(eventType EventType) *EventBuilder {
	return &EventBuilder{
		e: Event{
			Type:      eventType,
			Timestamp: time.Now().UnixNano(),
			Meta:      map[string]string{},
		},
	}
}

func (eb *EventBuilder) AddReason(reason string) *EventBuilder {
	eb.e.Meta["Reason"] = reason
	return eb
}

func (eb *EventBuilder) AddGoal(goal *ScienceGoal) *EventBuilder {
	eb.e.Meta["GoalName"] = goal.Name
	eb.e.Meta["GoalID"] = goal.ID
	return eb
}

func (eb *EventBuilder) AddPluginMeta(plugin *Plugin) *EventBuilder {
	eb.e.Meta["PluginName"] = plugin.Name
	eb.e.Meta["PluginImage"] = plugin.PluginSpec.Image
	eb.e.Meta["PluginStatus"] = string(plugin.Status.SchedulingStatus)
	eb.e.Meta["PluginTask"] = plugin.PluginSpec.Job
	eb.e.Meta["PluginArgs"] = strings.Join(plugin.PluginSpec.Args, " ")
	eb.e.Meta["PluginSelector"] = fmt.Sprint(plugin.PluginSpec.Selector)
	eb.e.Meta["GoalID"] = plugin.GoalID
	return eb
}

func (eb *EventBuilder) AddJobMeta(job *batchv1.Job) *EventBuilder {
	eb.e.Meta["JobName"] = job.Name
	if len(job.Status.Conditions) > 0 {
		eb.e.Meta["JobStatus"] = string(job.Status.Conditions[0].Type)
	}
	return eb
}

func (eb *EventBuilder) Build() Event {
	return eb.e
}

func (e *Event) get(name string) string {
	if value, ok := e.Meta[name]; ok {
		return value
	} else {
		return ""
	}
}

func (e *Event) GetGoalName() string {
	return e.get("GoalName")
}

func (e *Event) GetGoalID() string {
	return e.get("GoalID")
}

func (e *Event) GetPluginName() string {
	return e.get("PluginName")
}

func (e *Event) GetReason() string {
	return e.get("Reason")
}

func (e *Event) ToString() string {
	return string(e.Type)
}

// ToWaggleMessage converts an Event into a Waggle message that can be sent through Waggle infrastructure.
//
// A few points to make in this conversion is,
//
// - Topic name "sys.scheduler" will be used in converted Waggle message
//
// - Event.Type, Event.Body, and Event.Meta will be encoded into a JSON blob and be used as a Value in Waggle message
func (e *Event) ToWaggleMessage() *WaggleMessage {
	encodedBody, err := e.encodeToJson()
	if err != nil {
		logger.Debug.Printf("Failed to convert to Waggle message: %q", err.Error())
		return nil
	}
	return NewMessage(
		string(e.Type),
		encodedBody,
		e.Timestamp,
		map[string]string{},
	)
}

func (e *Event) encodeToJson() ([]byte, error) {
	body := map[string]interface{}{
		"Event": string(e.Type),
		"Meta":  e.Meta,
	}
	return json.Marshal(body)
}

type EventType string

const (
	EventSchedulingDecisionScheduled EventType = "sys.scheduler.decision.scheduled"
	EventGoalStatusNew               EventType = "sys.scheduler.status.goal.new"
	EventGoalStatusUpdated           EventType = "sys.scheduler.status.goal.updated"
	EventGoalStatusDeleted           EventType = "sys.scheduler.status.goal.deleted"
	EventPluginStatusPromoted        EventType = "sys.scheduler.status.plugin.promoted"
	EventPluginStatusScheduled       EventType = "sys.scheduler.status.plugin.scheduled"
	EventPluginStatusLaunched        EventType = "sys.scheduler.status.plugin.launched"
	EventPluginStatusComplete        EventType = "sys.scheduler.status.plugin.complete"
	EventPluginStatusFailed          EventType = "sys.scheduler.status.plugin.failed"
	EventFailure                     EventType = "sys.scheduler.failure"
)

// type EventErrorCode string

// const (
// 	CreateJob EventErrorCode = "100"
// )
