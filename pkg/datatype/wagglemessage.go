package datatype

import (
	b64 "encoding/base64"
	"encoding/json"
)

type WaggleMessage struct {
	Name      string            `json:"name"`
	Value     interface{}       `json:"val"`
	Timestamp int64             `json:"ts"`
	Meta      map[string]string `json:"meta"`
	Enc       string            `json:"enc"`
}

func NewMessage(name string, value interface{}, timestamp int64, meta map[string]string) *WaggleMessage {
	return &WaggleMessage{
		Name:      name,
		Value:     value,
		Timestamp: timestamp,
		Meta:      meta,
	}
}

func Dump(message *WaggleMessage) []byte {
	switch message.Value.(type) {
	case byte, []byte:
		message.Enc = "b64"
	}
	raw, _ := json.Marshal(message)
	return raw
}

func Load(raw []byte) *WaggleMessage {
	var message WaggleMessage
	json.Unmarshal(raw, &message)
	if message.Enc == "b64" {
		v, _ := b64.StdEncoding.DecodeString(message.Value.(string))
		message.Value = v
	}
	return &message
}
