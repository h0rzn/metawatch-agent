package stream

type Set struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewSet(setType string, data interface{}) *Set {
	return &Set{
		Type: setType,
		Data: data,
	}
}
