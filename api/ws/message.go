package ws

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewMessage(setType string, data interface{}) *Message {
	return &Message{
		Type: setType,
		Data: data,
	}
}
