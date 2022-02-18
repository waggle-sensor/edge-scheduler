package datatype

import (
	"fmt"
	"testing"
)

func TestEvent(t *testing.T) {
	newEvent := NewSimpleEvent(EventSchedulingDecision, "hello world")
	expected := `{"event":"sys.scheduler.decision","meta":{},"value":"hello world"}`
	// expected, err := json.marshal(map[string]interface{}{
	// 	"event": string(EventSchedulingDecision),
	// 	"meta": map[string]string{},
	// 	"value": "hello world",
	// })
	blob, err := newEvent.encodeToJson()
	if err != nil {
		t.Errorf("Error in encoding the event: %q", err.Error())
	} else {
		fmt.Printf("%s %v", expected, string(blob))
	}
}
