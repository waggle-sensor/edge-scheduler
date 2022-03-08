package datatype

import (
	"encoding/json"
	"fmt"
)

type APIMessageBuilder struct {
	message APIMessage
}

func NewAPIMessageBuilder() *APIMessageBuilder {
	return &APIMessageBuilder{
		message: APIMessage{
			body: make(map[string]interface{}),
		},
	}
}

func NewAPIMessageBuilderWithMessage(body map[string]interface{}) *APIMessageBuilder {
	return &APIMessageBuilder{
		message: APIMessage{
			body: body,
		},
	}
}

func (builder *APIMessageBuilder) AddError(reason string) *APIMessageBuilder {
	builder.AddEntity("error", reason)
	return builder
}

func (builder *APIMessageBuilder) AddEntity(key string, value interface{}) *APIMessageBuilder {
	builder.message.body[key] = value
	return builder
}

func (builder *APIMessageBuilder) Build() *APIMessage {
	return &builder.message
}

type APIMessage struct {
	body map[string]interface{}
}

func (m *APIMessage) ToJson() []byte {
	s, err := json.MarshalIndent(m.body, "", "  ")
	if err != nil {
		fmt.Printf("hack%s", err.Error())
		return nil
	}
	return s
}
