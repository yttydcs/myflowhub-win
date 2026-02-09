package transport

import (
	"encoding/json"
	"errors"
	"strings"
)

type Message struct {
	Action string `json:"action"`
	Data   any    `json:"data,omitempty"`
}

func EncodeMessage(action string, data any) ([]byte, error) {
	if strings.TrimSpace(action) == "" {
		return nil, errors.New("action is required")
	}
	return json.Marshal(Message{Action: action, Data: data})
}
