package datatype

// Message structs messages that handle events
type Message struct {
	EventPluginContext

	RequestToKB

	*ScienceGoal
}

// EventPluginContext structs a message about plugin context change
type EventPluginContext struct {
	Name   string
	Status ContextStatus
}

// ContextStatus represents contextual status of a plugin
type ContextStatus string

const (
	// Runnable indicates a plugin is runnable wrt the current context
	Runnable ContextStatus = "Runnable"
	// Stoppable indicates a plugin is stoppable wrt the current context
	Stoppable = "Stoppable"
)

// RequestToKB structs a message for KB
type RequestToKB struct {
	ReturnCode int         `json:"return_code"`
	Command    string      `json:"command"`
	Args       []string    `json:"args"`
	Result     interface{} `json:"result"`
}
