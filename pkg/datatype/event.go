package datatype

import (
	"encoding/json"
	"time"

	"github.com/sagecontinuum/ses/pkg/logger"
)

type Event struct {
	Type      EventType
	Body      string
	Timestamp int64
	Meta      map[string]string
}

func NewEvent(eventType EventType, body string, meta map[string]string) Event {
	return Event{
		Type:      eventType,
		Body:      body,
		Timestamp: time.Now().UnixNano(),
		Meta:      meta,
	}
}

func NewSimpleEvent(eventType EventType, body string) Event {
	return Event{
		Type:      eventType,
		Body:      body,
		Timestamp: time.Now().UnixNano(),
		Meta:      map[string]string{},
	}
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
		"event": string(e.Type),
		"value": e.Body,
		"meta":  e.Meta,
	}
	return json.Marshal(body)
}

type EventType string

const (
	EventSchedulingDecision   EventType = "sys.scheduler.decision"
	EventGoalStatusNew        EventType = "sys.scheduler.status.goal.new"
	EventGoalStatusUpdated    EventType = "sys.scheduler.status.goal.update"
	EventGoalStatusDeleted    EventType = "sys.scheduler.status.goal.deleted"
	EventPluginStatusLaunched EventType = "sys.scheduler.status.plugin.launched"
	EventPluginStatusComplete EventType = "sys.scheduler.status.plugin.complete"
	EventPluginStatusFailed   EventType = "sys.scheduler.status.plugin.failed"
	EventFailure              EventType = "sys.scheduler.failure"
)

// type EventErrorCode string

// const (
// 	CreateJob EventErrorCode = "100"
// )
