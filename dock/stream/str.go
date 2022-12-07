package stream

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ticker duration for interv receivers
const IntervDur time.Duration = 5 * time.Second

type Str struct {
	mutex   *sync.RWMutex
	Src     io.ReadCloser
	Source  Source
	Strg    *Storage
	closing bool
}

type Source struct {
	R     io.ReadCloser
	Done  chan struct{}
	Dried chan struct{}
}

type Storage struct {
	mutex     *sync.RWMutex
	Receivers map[*Receiver]bool
	Disabled  map[*Receiver]bool
	LveC      chan *Receiver
}

func (st *Storage) AddRcv(interv bool) *Receiver {
	r := NewReceiver(interv, st.LveC)
	st.mutex.Lock()
	st.Receivers[r] = true
	st.mutex.Unlock()
	return r
}

func (st *Storage) Disable(lvrs ...*Receiver) {
	st.mutex.Lock()
	for _, lvr := range lvrs {
		st.Disabled[lvr] = true
	}
	st.mutex.Unlock()
}

func (st *Storage) Rm(r *Receiver) {
	fmt.Println("rm reciever")
	r.Close(true)
	fmt.Println("closed r")
	delete(st.Receivers, r)
}

func (st *Storage) DisabelAll() {
	st.mutex.Lock()
	for rcv := range st.Receivers {
		st.Disabled[rcv] = true
	}
	st.mutex.Unlock()
}

func (st *Storage) Cleanup() bool {
	st.mutex.Lock()
	for rcv := range st.Disabled {
		st.Rm(rcv)
	}
	for dbd := range st.Disabled {
		delete(st.Disabled, dbd)
	}
	empty := len(st.Receivers) == 0
	st.mutex.Unlock()
	return empty
}

func NewStr(src io.ReadCloser) *Str {
	return &Str{
		mutex: &sync.RWMutex{},
		Source: Source{
			R:     src,
			Done:  make(chan struct{}),
			Dried: make(chan struct{}),
		},
		Strg: &Storage{
			mutex:     &sync.RWMutex{},
			Receivers: make(map[*Receiver]bool),
			Disabled:  make(map[*Receiver]bool),
			LveC:      make(chan *Receiver),
		},
		closing: false,
	}
}

func (s *Str) Join(interv bool) (*Receiver, error) {
	if s.closing {
		return &Receiver{}, errors.New("cant join: streamer closing")
	}
	rcv := s.Strg.AddRcv(interv)
	return rcv, nil
}

func (s *Str) Close() error {
	fmt.Println("closing streamer")
	s.closing = true
	s.Source.Done <- struct{}{}
	for rcv := range s.Strg.Receivers {
		s.Strg.Rm(rcv)
	}
	select {
	case <-s.Source.Dried:
		fmt.Println("dry out sig received")
		return nil
	case <-time.After(15 * time.Second):
		return errors.New("data pipeline failed to dry out")
	}
}

type GenPipe func(r io.ReadCloser, done chan struct{}, dried chan struct{}) chan Set

func (s *Str) Run(pipe GenPipe) {
	sets := pipe(s.Source.R, s.Source.Done, s.Source.Dried)

	intervTick := time.NewTicker(IntervDur)
	logrus.Debugln("- STREAMER - pipeline succesfully built")

	defer logrus.Debugln("- STREAMER - exited")
	for {
		select {
		case lvr := <-s.Strg.LveC:
			fmt.Println("disabling")
			s.Strg.Disable(lvr)
		case set, ok := <-sets:
			if !ok {
				if !s.closing {
					s.Close()
				}
				return
			}
			for recv := range s.Strg.Receivers {
				if recv.Interv {
					select {
					case <-intervTick.C:
						recv.In <- set
					default:
					}
				} else {
					recv.In <- set
				}
			}

			if s.Strg.Cleanup() {
				fmt.Println("empty: closing")
				s.Close()
				return
			}
		}
	}
}
