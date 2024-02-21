package datatype

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
)

type Event interface {
}

type SchedulerEventBuilder struct {
	e SchedulerEvent
}

type SchedulerEvent struct {
	Type      EventType
	Timestamp int64
	Meta      map[string]interface{}
}

func NewSchedulerEventBuilder(eventType EventType) *SchedulerEventBuilder {
	return &SchedulerEventBuilder{
		e: SchedulerEvent{
			Type:      eventType,
			Timestamp: time.Now().UnixNano(),
			Meta:      map[string]interface{}{},
		},
	}
}

func (s *SchedulerEventBuilder) AddValue(v interface{}) *SchedulerEventBuilder {
	s.e.Meta["value"] = v
	return s
}

func (s *SchedulerEventBuilder) AddReason(reason string) *SchedulerEventBuilder {
	s.e.Meta["reason"] = reason
	return s
}

func (s *SchedulerEventBuilder) AddJob(j *Job) *SchedulerEventBuilder {
	s.e.Meta["job_id"] = j.JobID
	return s
}

func (s *SchedulerEventBuilder) AddGoal(goal *ScienceGoal) *SchedulerEventBuilder {
	s.e.Meta["goal_name"] = goal.Name
	s.e.Meta["goal_id"] = goal.ID
	return s
}

func (s *SchedulerEventBuilder) AddEntry(k string, v interface{}) *SchedulerEventBuilder {
	s.e.Meta[k] = v
	return s
}

func (s *SchedulerEventBuilder) AddPluginMeta(plugin Plugin) *SchedulerEventBuilder {
	s.e.Meta["plugin_name"] = plugin.Name
	s.e.Meta["plugin_image"] = plugin.PluginSpec.Image
	s.e.Meta["plugin_task"] = plugin.PluginSpec.Job
	s.e.Meta["plugin_args"] = strings.Join(plugin.PluginSpec.Args, " ")
	selectors, err := json.Marshal(plugin.PluginSpec.Selector)
	if err == nil {
		s.e.Meta["plugin_selector"] = string(selectors)
	}
	s.e.Meta["goal_id"] = plugin.GoalID
	return s
}

func (s *SchedulerEventBuilder) AddK3SJobMeta(job *batchv1.Job) *SchedulerEventBuilder {
	if job == nil {
		return s
	}
	s.e.Meta["k3s_job_name"] = job.Name
	if len(job.Status.Conditions) > 0 {
		s.e.Meta["k3s_job_status"] = string(job.Status.Conditions[0].Type)
		// job.Status.Conditions[0].
	}
	return s
}

func (s *SchedulerEventBuilder) AddPodMeta(pod *apiv1.Pod) *SchedulerEventBuilder {
	if pod == nil {
		return s
	}
	s.e.Meta["k3s_pod_name"] = pod.Name
	s.e.Meta["k3s_pod_status"] = string(pod.Status.Phase)
	s.e.Meta["k3s_pod_node_name"] = pod.Spec.NodeName
	if v, found := pod.Labels["sagecontinuum.org/plugin-instance"]; found {
		s.e.Meta["k3s_pod_instance"] = v
	}
	return s
}

func (s *SchedulerEventBuilder) Build() Event {
	return s.e
}

func (e *SchedulerEvent) get(name string) interface{} {
	if value, ok := e.Meta[name]; ok {
		return value
	} else {
		return ""
	}
}

func (e *SchedulerEvent) GetJobID() string {
	return e.get("job_id").(string)
}

func (e *SchedulerEvent) GetGoalName() string {
	return e.get("goal_name").(string)
}

func (e *SchedulerEvent) GetGoalID() string {
	return e.get("goal_id").(string)
}

func (e *SchedulerEvent) GetPluginName() string {
	return e.get("plugin_name").(string)
}

func (e *SchedulerEvent) GetReason() string {
	return e.get("reason").(string)
}

func (e *SchedulerEvent) GetEntry(k string) interface{} {
	return e.get(k)
}

func (e *SchedulerEvent) ToString() string {
	return string(e.Type)
}

func NewSchedulerEventBuilderFromWaggleMessage(m *WaggleMessage) (*SchedulerEventBuilder, error) {
	builder := NewSchedulerEventBuilder(EventType(m.Name))
	builder.e.Timestamp = m.Timestamp
	var body map[string]interface{}
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
func (e *SchedulerEvent) ToWaggleMessage() *WaggleMessage {
	// TODO: beehive-influxdb does not handle bytes so body is always string.
	//       This should be lifted once it accepts bytes.
	body := e.get("value")
	if body == "" {
		encodedBody, err := e.EncodeMetaToJson()
		if err != nil {
			logger.Debug.Printf("Failed to convert to Waggle message: %q", err.Error())
			return nil
		}
		body = string(encodedBody)
	}
	return NewMessage(
		string(e.Type),
		body,
		e.Timestamp,
		map[string]string{},
	)
}

func (e *SchedulerEvent) EncodeMetaToJson() ([]byte, error) {
	return json.Marshal(e.Meta)
}

type EventType string

const (
	EventRabbitMQSubscriptionPatternAll     string = "sys.scheduler.#"
	EventRabbitMQSubscriptionPatternGoals   string = "sys.scheduler.status.goal.#"
	EventRabbitMQSubscriptionPatternPlugins string = "sys.scheduler.status.plugin.#"
	// EventSchedulingDecisionScheduled EventType = "sys.scheduler.decision.scheduled"
	EventJobStatusSuspended     EventType = "sys.scheduler.status.job.suspended"
	EventJobStatusRemoved       EventType = "sys.scheduler.status.job.removed"
	EventGoalStatusSubmitted    EventType = "sys.scheduler.status.goal.submitted"
	EventGoalStatusUpdated      EventType = "sys.scheduler.status.goal.updated"
	EventGoalStatusReceived     EventType = "sys.scheduler.status.goal.received"
	EventGoalStatusReceivedBulk EventType = "sys.scheduler.status.goal.received.bulk"
	EventGoalStatusRemoved      EventType = "sys.scheduler.status.goal.removed"

	EventPluginStatusQueued    EventType = "sys.scheduler.status.plugin.queued"
	EventPluginStatusSelected  EventType = "sys.scheduler.status.plugin.selected"
	EventPluginStatusScheduled EventType = "sys.scheduler.status.plugin.scheduled"
	EventPluginStatusRunning   EventType = "sys.scheduler.status.plugin.running"
	EventPluginStatusLaunched  EventType = "sys.scheduler.status.plugin.launched"
	EventPluginStatusCompleted EventType = "sys.scheduler.status.plugin.completed"
	EventPluginLastExecution   EventType = "sys.scheduler.plugin.lastexecution"
	EventPluginStatusFailed    EventType = "sys.scheduler.status.plugin.failed"
	EventFailure               EventType = "sys.scheduler.failure"

	EventPluginPerfCPU EventType = "sys.plugin.perf.cpu"
	EventPluginPerfMem EventType = "sys.plugin.perf.mem"
	EventPluginPerfGPU EventType = "sys.plugin.perf.gpu"
)

// type EventErrorCode string

// const (
// 	CreateJob EventErrorCode = "100"
// )
