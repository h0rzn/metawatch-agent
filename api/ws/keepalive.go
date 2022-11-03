package ws

import (
	"fmt"
	"time"
)

type KeepAlive struct {
	Interv    time.Duration
	Challenge chan bool
	Done      chan bool
	Timer     *time.Timer
}

func NewKeepAlive(dur time.Duration) *KeepAlive {
	return &KeepAlive{
		Interv:    dur,
		Challenge: make(chan bool),
		Done:      make(chan bool),
		Timer:     time.NewTimer(dur),
	}
}

func (k *KeepAlive) Reset() {
	if !k.Timer.Stop() {
		<-k.Timer.C
	}
	k.Timer.Reset(k.Interv)
}

func (k *KeepAlive) Run() {
	for {
		select {
		case <-k.Timer.C:
			fmt.Println("timer triggered")
			k.Challenge <- true
		case <-k.Done:
			k.Timer.Stop()
			return
		}

	}
}
