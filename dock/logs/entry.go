package logs

type Entry struct {
	Time string `json:"when"`
	Type string `json:"type"`
	Data string `json:"data"`
}

func NewEntry(time, data string, typ byte) *Entry {
	var streamType string
	if typ == 1 {
		streamType = "stdout"
	} else {
		streamType = "stderr"
	}

	return &Entry{
		Time: time,
		Type: streamType,
		Data: data,
	}
}
