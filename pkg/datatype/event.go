package datatype

type Event struct {
	Type EventType
	Meta string
}

func NewEvent(eventType EventType, meta string) Event {
	return Event{
		Type: eventType,
		Meta: meta,
	}
}

type EventType string

const (
	EventNewGoal            EventType = "EventNewGoal"
	EventPluginStatusChange EventType = "EventPluginStatusChange"
)
