package ws

import "encoding/json"

type Message struct {
	Type string      `json:"type"`
	Set  interface{} `json:"set"`
}

func NewMessage(setType string, set interface{}) ([]byte, error) {
	msg := &Message{
		Type: setType,
		Set:  set,
	}
	return json.Marshal(msg)
}
