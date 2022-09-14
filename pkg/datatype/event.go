package datatype

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
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
	eb.e.Meta["reason"] = reason
	return eb
}

func (eb *EventBuilder) AddJob(j *Job) *EventBuilder {
	eb.e.Meta["job_id"] = j.JobID
	return eb
}

func (e *Event) GetJobID() string {
	return e.get("job_id")
}

func (eb *EventBuilder) AddGoal(goal *ScienceGoal) *EventBuilder {
	eb.e.Meta["goal_name"] = goal.Name
	eb.e.Meta["goal_id"] = goal.ID
	return eb
}

func (eb *EventBuilder) AddEntry(k string, v string) *EventBuilder {
	eb.e.Meta[k] = v
	return eb
}

func (eb *EventBuilder) AddPluginMeta(plugin *Plugin) *EventBuilder {
	eb.e.Meta["plugin_name"] = plugin.Name
	eb.e.Meta["plugin_image"] = plugin.PluginSpec.Image
	eb.e.Meta["plugin_status_by_scheduler"] = string(plugin.Status.SchedulingStatus)
	eb.e.Meta["plugin_task"] = plugin.PluginSpec.Job
	eb.e.Meta["plugin_args"] = strings.Join(plugin.PluginSpec.Args, " ")
	selectors, err := json.Marshal(plugin.PluginSpec.Selector)
	if err == nil {
		eb.e.Meta["plugin_selector"] = string(selectors)
	}
	eb.e.Meta["goal_id"] = plugin.GoalID
	return eb
}

func (eb *EventBuilder) AddK3SJobMeta(job *batchv1.Job) *EventBuilder {
	if job == nil {
		return eb
	}
	eb.e.Meta["k3s_job_name"] = job.Name
	if len(job.Status.Conditions) > 0 {
		eb.e.Meta["k3s_job_status"] = string(job.Status.Conditions[0].Type)
		// job.Status.Conditions[0].
	}
	return eb
}

func (eb *EventBuilder) AddPodMeta(pod *apiv1.Pod) *EventBuilder {
	if pod == nil {
		return eb
	}
	eb.e.Meta["k3s_pod_name"] = pod.Name
	eb.e.Meta["k3s_pod_status"] = string(pod.Status.Phase)
	eb.e.Meta["k3s_pod_node_name"] = pod.Spec.NodeName
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
	return e.get("goal_name")
}

func (e *Event) GetGoalID() string {
	return e.get("goal_id")
}

func (e *Event) GetPluginName() string {
	return e.get("plugin_name")
}

func (e *Event) GetReason() string {
	return e.get("reason")
}

func (e *Event) GetEntry(k string) string {
	return e.get(k)
}

func (e *Event) ToString() string {
	return string(e.Type)
}

func NewEventBuilderFromWaggleMessage(m *WaggleMessage) (*EventBuilder, error) {
	builder := NewEventBuilder(EventType(m.Name))
	builder.e.Timestamp = m.Timestamp
	var body map[string]string
	err := json.Unmarshal([]byte(m.Value.(string)), &body)
	if err != nil {
		return nil, err
	}
	builder.e.Meta = body
	return builder, nil
}

// ToWaggleMessage converts an Event into a Waggle message that can be sent through Waggle infrastructure.
//
// A few points to make in this conversion is,
//
// - Topic name "sys.scheduler" will be used in converted Waggle message
//
// - Event.Type, Event.Body, and Event.Meta will be encoded into a JSON blob and be used as a Value in Waggle message
func (e *Event) ToWaggleMessage() *WaggleMessage {
	// TODO: beehive-influxdb does not handle bytes so body is always string.
	//       This should be lifted once it accepts bytes.
	encodedBody, err := e.encodeMetaToJson()
	if err != nil {
		logger.Debug.Printf("Failed to convert to Waggle message: %q", err.Error())
		return nil
	}
	return NewMessage(
		string(e.Type),
		string(encodedBody),
		e.Timestamp,
		map[string]string{},
	)
}

func (e *Event) encodeMetaToJson() ([]byte, error) {
	return json.Marshal(e.Meta)
}

type EventType string

const (
	// EventSchedulingDecisionScheduled EventType = "sys.scheduler.decision.scheduled"
	EventJobStatusSuspended     EventType = "sys.scheduler.status.job.suspended"
	EventJobStatusRemoved       EventType = "sys.scheduler.status.job.removed"
	EventGoalStatusSubmitted    EventType = "sys.scheduler.status.goal.submitted"
	EventGoalStatusUpdated      EventType = "sys.scheduler.status.goal.updated"
	EventGoalStatusReceived     EventType = "sys.scheduler.status.goal.received"
	EventGoalStatusReceivedBulk EventType = "sys.scheduler.status.goal.received.bulk"
	EventGoalStatusRemoved      EventType = "sys.scheduler.status.goal.removed"
	EventPluginStatusPromoted   EventType = "sys.scheduler.status.plugin.promoted"
	EventPluginStatusScheduled  EventType = "sys.scheduler.status.plugin.scheduled"
	EventPluginStatusLaunched   EventType = "sys.scheduler.status.plugin.launched"
	EventPluginStatusComplete   EventType = "sys.scheduler.status.plugin.complete"
	EventPluginLastExecution    EventType = "sys.scheduler.plugin.lastexecution"
	EventPluginStatusFailed     EventType = "sys.scheduler.status.plugin.failed"
	EventFailure                EventType = "sys.scheduler.failure"
)

// type EventErrorCode string

// const (
// 	CreateJob EventErrorCode = "100"
// )
