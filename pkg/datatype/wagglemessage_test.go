package datatype

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestNewWaggleMessage(t *testing.T) {
	name := "new"
	value := 12345
	timestamp := time.Now().UnixNano()
	meta := map[string]string{}
	message := WaggleMessage{
		Name:      name,
		Value:     value,
		Timestamp: timestamp,
		Meta:      meta,
	}
	newWaggleMessage := NewMessage(name, value, timestamp, meta)
	if !reflect.DeepEqual(message, *newWaggleMessage) {
		t.Errorf("NewMessage failed to return the correct message")
	}
}

func TestWaggleMessageValueTypes(t *testing.T) {
	tests := map[string]struct {
		message *WaggleMessage
	}{
		"IntMessage": {
			message: NewMessage(
				"test.int",
				float64(12345), // Json unmarshal considers numbers as float64
				time.Now().UnixNano(),
				map[string]string{}),
		},
		"FloatMessage": {
			message: NewMessage(
				"test.float",
				float64(12345.54321),
				time.Now().UnixNano(),
				map[string]string{}),
		},
		"StringMessage": {
			message: NewMessage(
				"test.string",
				"hello world",
				time.Now().UnixNano(),
				map[string]string{}),
		},
		"BytesMessage": {
			message: NewMessage(
				"test.string",
				[]byte("hello world"),
				time.Now().UnixNano(),
				map[string]string{}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			originalMessage := NewMessage(tc.message.Name, tc.message.Value, tc.message.Timestamp, tc.message.Meta)
			encodedMessage := Dump(originalMessage)
			decodedMessage, _ := Load(encodedMessage)
			switch originalMessage.Value.(type) {
			case float64:
				if originalMessage.Value != decodedMessage.Value {
					t.Errorf("Value not match: original %f, but returned %f", originalMessage.Value, decodedMessage.Value)
				}
			case string:
				if originalMessage.Value != decodedMessage.Value {
					t.Errorf("Value not match: original %q, but returned %q", originalMessage.Value, decodedMessage.Value)
				}
			case byte, []byte:
				if bytes.Compare(originalMessage.Value.([]byte), decodedMessage.Value.([]byte)) != 0 {
					t.Errorf("Value not match: original %v, but returned %v", originalMessage.Value, decodedMessage.Value)
				}
			}
		})
	}
}
