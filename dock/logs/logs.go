package logs

import (
	"encoding/binary"
	"io"
	"strings"
)

type Logs struct {
	Reader struct {
		Active bool
		R      io.Reader
	}
	Streamer Streamer

	// "active" / "inactive"
	// "refreshing" indicates that refreshing is in action and no
	// new refreshes should be called
	State   string
	Refresh chan io.Reader
}

type Streamer struct {
	Reg  chan Consumer
	Ureg chan Consumer
	Dis  chan *Entry
	Done chan bool
}

type Consumer struct {
	Get  chan Entry
	Done chan bool
}

type Entry struct {
	Time string `json:"when"`
	Type string `json:"type"`
	Data string `json:"data"`
}

func (s *Streamer) Request()

func (s *Streamer) Parse(d chan bool, r io.Reader) (<-chan Entry, <-chan error) {
	out := make(chan Entry)
	errChan := make(chan error)

	hdr := make([]byte, 8)
	for {
		select {
		case <-d:
			close(out)
			close(errChan)
		default:
		}

		sizes := binary.BigEndian.Uint32(hdr[:4])
		content := make([]byte, sizes)
		_, err := r.Read(content)
		if err != nil && err != io.EOF {
			errChan <- err
		}

		time, data, found := strings.Cut(string(content), " ")
		if found {
			var streamType string
			if hdr[0] == 1 {
				streamType = "stdout"
			} else {
				streamType = "stderr"
			}

			out <- Entry{
				Time: time,
				Type: streamType,
				Data: data,
			}
		}
	}
}

func (s *Streamer) Run() {

	go func(done chan bool) {
		// distribute
	}(s.Done)

	for {
		select {
		case c := <-s.Reg:
			_ = c
		case c := <-s.Ureg:
			_ = c
		case <-s.Done:
			// handle shutdown of streamer
		}
	}
}
