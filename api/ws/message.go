package ws

import "encoding/json"

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewMessage(setType string, data interface{}) ([]byte, error) {
	msg := &Message{
		Type: setType,
		Data: data,
	}
	return json.Marshal(msg)
}
