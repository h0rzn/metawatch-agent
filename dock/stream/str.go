package stream

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// ticker duration for interv receivers
const IntervDur time.Duration = 5 * time.Second

type Str struct {
	mutex   *sync.RWMutex
	Src     io.ReadCloser
	Pipe    Pipeline
	Strg    *Storage
	Closing bool
	Closed  chan error
}

type Storage struct {
	mutex     *sync.RWMutex
	Receivers map[*Receiver]bool
	LveC      chan *Receiver
}

func (st *Storage) Add(interv bool) *Receiver {
	r := NewReceiver(interv, st.LveC)
	st.mutex.Lock()
	st.Receivers[r] = true
	st.mutex.Unlock()
	return r
}

func (st *Storage) Cls(rcv *Receiver) {
	st.mutex.Lock()
	if _, exists := st.Receivers[rcv]; exists {
		rcv.Close(true)
		close(rcv.In)
		delete(st.Receivers, rcv)
	}
	st.mutex.Unlock()
}

func NewStr(pipeline Pipeline) *Str {
	return &Str{
		mutex: &sync.RWMutex{},
		Pipe:  pipeline,
		Strg: &Storage{
			mutex:     &sync.RWMutex{},
			Receivers: make(map[*Receiver]bool),
			LveC:      make(chan *Receiver),
		},
		Closing: false,
	}
}

func (s *Str) Join(interv bool) (*Receiver, error) {
	if s.Closing {
		return &Receiver{}, errors.New("cant join: streamer closing")
	}
	rcv := s.Strg.Add(interv)
	return rcv, nil
}

func (s *Str) Cls() <-chan error {
	closed := make(chan error, 1)
	s.Closing = true

	go func() {
		s.Pipe.Stop()
		for r := range s.Strg.Receivers {
			s.Strg.Cls(r)
		}
		fmt.Println("closed all receivers")
		close(s.Strg.LveC)
		closed <- nil
		fmt.Println("closed")
	}()

	return closed
}

func (s *Str) Run() {
	data := s.Pipe.Out()

	go func() {
		for lvr := range s.Strg.LveC {
			if s.Closing {
				continue
			}
			fmt.Println("str: handling leaver")
			s.Strg.Cls(lvr)

			if len(s.Strg.Receivers) == 0 {
				fmt.Println("no receivers left")
			}

		}
	}()

	for set := range data {
		for recv := range s.Strg.Receivers {
			// send if channel is empty
			select {
			case recv.In <- set:
			default:
			}
		}
	}
	// we can exit now

}
