package datatype

import (
	"reflect"
	"testing"
)

func TestEventWaggleConversion(t *testing.T) {
	tests := map[string]struct {
		Type    string
		Payload map[string]string
	}{
		"simple": {
			Type: string(EventPluginStatusLaunched),
			Payload: map[string]string{
				"test": "great",
			},
		},
	}
	for _, test := range tests {
		e := NewEventBuilder(EventType(test.Type))
		for k, v := range test.Payload {
			e.AddEntry(k, v)
		}
		msg := e.Build()
		waggleMsg := msg.ToWaggleMessage()
		unWaggleMsgBuilder, _ := NewEventBuilderFromWaggleMessage(waggleMsg)
		unWaggleMsg := unWaggleMsgBuilder.Build()
		if unWaggleMsg.Type != EventType(test.Type) {
			t.Errorf("Type mismatch: wanted %s, got %s", test.Type, unWaggleMsg.Type)
		}
		if !reflect.DeepEqual(test.Payload, unWaggleMsg.Meta) {
			t.Errorf("Type mismatch: wanted %v, got %v", test.Payload, unWaggleMsg.Meta)
		}
	}
}
