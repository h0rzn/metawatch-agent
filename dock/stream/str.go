package stream

import (
	"fmt"
	"io"
	"sync"
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
	fmt.Println("[STREAMER] leaving recv")
	s.mutex.Lock()
	// handle receiver side quit
	delete(s.Receivers, recv)
	s.mutex.Unlock()

	if len(s.Receivers) == 0 {
		s.Exit()
	}
}

func (s *Str) Exit() {
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
	fmt.Println("[STREAMER] pipeline succesfully built")

	// receiver leave events
	go func() {
		fmt.Println("[STREAMER] listening for recvleave")
		for leaver := range s.RecvLeave {
			fmt.Println("[STREAMER] handling recv leave")
			s.Leave(leaver)
			if len(s.Receivers) == 0 {
				s.Exit()
			}
		}
		fmt.Println("[STREAMER] stopped recleave listener")
	}()

	// relay data to receivers
	for set := range sets {
		s.mutex.RLock()
		for recv := range s.Receivers {
			recv.In <- set
		}
		s.mutex.RUnlock()
	}

	s.Exit()
}
