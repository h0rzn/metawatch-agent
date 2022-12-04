package stream

import (
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
	SrcDone chan struct{}
	Strg    *Storage
	Quit    chan struct{}
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

func (st *Storage) AddLvr(lvrs ...*Receiver) {
	st.mutex.Lock()
	for _, lvr := range lvrs {
		st.Disabled[lvr] = true
	}
	st.mutex.Unlock()
}

func (st *Storage) Rm(r *Receiver) {
	st.mutex.Lock()
	r.Quit()
	delete(st.Receivers, r)
	delete(st.Disabled, r)
	st.mutex.Unlock()
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
	st.mutex.Unlock()
	return len(st.Receivers) == 0
}

func NewStr(src io.ReadCloser) *Str {
	return &Str{
		mutex:   &sync.RWMutex{},
		Src:     src,
		SrcDone: make(chan struct{}),
		Quit:    make(chan struct{}),
		Strg: &Storage{
			mutex:     &sync.RWMutex{},
			Receivers: make(map[*Receiver]bool),
			Disabled:  make(map[*Receiver]bool),
			LveC:      make(chan *Receiver),
		},
	}
}

func (s *Str) Join(interv bool) *Receiver {
	return s.Strg.AddRcv(interv)
}

func (s *Str) Close() {
	fmt.Println("closing streamer")
	s.mutex.Lock()
	s.SrcDone <- struct{}{}
	s.mutex.Unlock()
}

func (s *Str) Exit() {
	s.Strg.DisabelAll()
}

type GenPipe func(r io.ReadCloser, done chan struct{}) <-chan Set

func (s *Str) Run(pipe GenPipe) {
	s.SrcDone = make(chan struct{})
	sets := pipe(s.Src, s.SrcDone)
	intervTick := time.NewTicker(IntervDur)
	logrus.Debugln("- STREAMER - pipeline succesfully built")

	go func() {
		logrus.Debug("- STREAMER - listening for recvleave")
		for lvr := range s.Strg.LveC {
			logrus.Debugln("- STREAMER - handling recv leave")
			s.Strg.AddLvr(lvr)
		}
	}()

	for set := range sets {
		s.Strg.mutex.Lock()
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
		s.Strg.mutex.Unlock()

		if s.Strg.Cleanup() {
			// no receivers left
			break
		}
	}

	s.Close()
	logrus.Debugln("- STREAMER - exited")
}
