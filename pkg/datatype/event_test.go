package datatype

import (
	"reflect"
	"testing"
)

func TestEventWaggleConversion(t *testing.T) {
	tests := map[string]struct {
		Type    string
		Payload map[string]interface{}
	}{
		"simple": {
			Type: string(EventPluginStatusLaunched),
			Payload: map[string]interface{}{
				"test":  "great",
				"float": 3.14,
			},
		},
	}
	for _, test := range tests {
		e := NewSchedulerEventBuilder(EventType(test.Type))
		for k, v := range test.Payload {
			switch v.(type) {
			case string:
				e.AddEntry(k, v.(string))
			case int:
				t.Errorf("integer type is not supported. use float instead.")
			case float64:
				e.AddEntry(k, v.(float64))
			}
		}
		msg := e.Build().(SchedulerEvent)
		waggleMsg := msg.ToWaggleMessage()
		unWaggleMsgBuilder, _ := NewSchedulerEventBuilderFromWaggleMessage(waggleMsg)
		unWaggleMsg := unWaggleMsgBuilder.Build().(SchedulerEvent)
		if unWaggleMsg.Type != EventType(test.Type) {
			t.Errorf("Type mismatch: wanted %s, got %s", test.Type, unWaggleMsg.Type)
		}
		if !reflect.DeepEqual(test.Payload, unWaggleMsg.Meta) {
			t.Errorf("Type mismatch: wanted %v, got %v", test.Payload, unWaggleMsg.Meta)
		}
	}
}
