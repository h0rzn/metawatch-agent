package stream

import (
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

type Str struct {
	mutex     *sync.RWMutex
	Src       io.Reader
	SrcDone   chan struct{}
	Receivers map[*Receiver]bool
	RecvLeave chan *Receiver
	Quit      chan struct{}
}

func NewStr(src io.Reader) *Str {
	return &Str{
		mutex:     &sync.RWMutex{},
		Src:       src,
		SrcDone:   make(chan struct{}),
		Receivers: make(map[*Receiver]bool),
		RecvLeave: make(chan *Receiver),
		Quit:      make(chan struct{}),
	}
}

func (s *Str) Join() *Receiver {
	r := NewReceiver(s.RecvLeave)
	s.mutex.Lock()
	s.Receivers[r] = true
	s.mutex.Unlock()
	return r
}

func (s *Str) Leave(recv *Receiver) {
	logrus.Debugln("- STREAMER - receiver leaving")
	s.mutex.Lock()
	// handle receiver side quit
	delete(s.Receivers, recv)
	s.mutex.Unlock()

	if len(s.Receivers) == 0 {
		s.Exit()
	}
}

func (s *Str) Exit() {
	logrus.Debugln("- Streamer - exit...")
	s.mutex.Lock()
	for recv := range s.Receivers {
		s.Leave(recv)
	}
	s.mutex.Unlock()
	// close reader?

	s.SrcDone <- struct{}{}
	s.Quit <- struct{}{}

}

type GenPipe func(r io.Reader, done chan struct{}) <-chan Set

func (s *Str) Run(pipe GenPipe) {
	s.SrcDone = make(chan struct{})
	sets := pipe(s.Src, s.SrcDone)
	logrus.Debugln("- STREAMER - pipeline succesfully built")

	// receiver leave events
	go func() {
		logrus.Debug("- STREAMER - listening for recvleave")
		for leaver := range s.RecvLeave {
			logrus.Debugln("- STREAMER - handling recv leave")
			s.Leave(leaver)
			if len(s.Receivers) == 0 {
				s.Exit()
			}
		}
		logrus.Debugln("- STREAMER - stopped recleave listener")
	}()

	// relay data to receivers
	for set := range sets {
		s.mutex.RLock()
		for recv := range s.Receivers {
			recv.In <- set
		}
		s.mutex.RUnlock()
	}
	logrus.Debugln("- STREAMER - no longer receiving data, preparing exit")

	s.Exit()
}
