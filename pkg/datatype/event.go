package datatype

import "time"

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

func (e *Event) ToWaggleMessage() *WaggleMessage {
	return NewMessage(
		string(e.Type),
		e.Body,
		e.Timestamp,
		e.Meta,
	)
}

type EventType string

const (
	EventNewGoal              EventType = "sys.scheduler.newgoal"
	EventPluginStatusLaunched EventType = "sys.scheduler.status.plugin.launched"
	EventPluginStatusComplete EventType = "sys.scheduler.status.plugin.complete"
	EventPluginStatusFailed   EventType = "sys.scheduler.status.plugin.failed"
	EventFailure              EventType = "sys.scheduler.failure"
)

// type EventErrorCode string

// const (
// 	CreateJob EventErrorCode = "100"
// )
