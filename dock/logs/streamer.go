package logs

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Streamer struct {
	mutex *sync.Mutex
	r     io.Reader
	Subs  map[stream.Subscriber]bool
	Done  chan struct{}
}

func NewStreamer(r io.Reader) *Streamer {
	return &Streamer{
		mutex: &sync.Mutex{},
		r:     r,
		Subs:  make(map[stream.Subscriber]bool),
		Done:  make(chan struct{}),
	}
}

// Quit completely shutdowns streamer and quits all readers
func (s *Streamer) Quit() {
	s.mutex.Lock()
	for sub := range s.Subs {
		sub.Quit()
	}
	s.mutex.Unlock()
}

// Subscribe creates new subscriber and adds them
func (s *Streamer) Subscribe() stream.Subscriber {
	sub := stream.NewTempSub()
	fmt.Println("[SUB]")
	s.mutex.Lock()
	s.Subs[sub] = true
	s.mutex.Unlock()
	return sub
}

// Unsubscribe quits the subscriber and removes them
// if no subscribers are left, the streamer quits
func (s *Streamer) Unsubscribe(sub stream.Subscriber) {
	fmt.Println("[USUB]")
	s.mutex.Lock()
	sub.Quit()
	delete(s.Subs, sub)
	if len(s.Subs) == 0 {
		fmt.Println("no subs left: closing streamer")
		s.Quit()
	}
	s.mutex.Unlock()
}

// parse parses the byte object to a log entry
// it returns the live channel of results and an error channel
func (s *Streamer) parse(d chan struct{}) (<-chan stream.Set, <-chan error) {
	out := make(chan stream.Set)
	errChan := make(chan error)

	go func() {
		for {
			hdr := make([]byte, 8)

			select {
			case <-d:
				close(out)
				close(errChan)
			default:
			}

			_, err := s.r.Read(hdr)
			if err != nil {
				if err == io.EOF {
					fmt.Println("eof")
				} else {
					errChan <- err
				}
			}

			sizes := binary.BigEndian.Uint32(hdr[4:])
			content := make([]byte, sizes)
			_, err = s.r.Read(content)
			_ = err
			if err != nil && err != io.EOF {
				errChan <- err
			}
			time, data, found := strings.Cut(string(content), " ")
			if found {
				entry := NewEntry(time, data, hdr[0])
				set := stream.NewSet("log_entry", entry)
				out <- *set
			}
		}
	}()
	return out, errChan
}

// Run inits parsing and relays entries to the subs
func (s *Streamer) Run() {
	done := make(chan struct{})
	logs, errChan := s.parse(done)
	fmt.Println("parser running")

	for {
		select {
		case entry := <-logs:
			if len(s.Subs) > 0 {
				for sub := range s.Subs {
					sub.Put(stream.NewSet("log_entry", entry))
				}
			} else {
				s.Quit()
			}
		case err := <-errChan:
			fmt.Println("parse err", err.Error())
		}
	}

}
