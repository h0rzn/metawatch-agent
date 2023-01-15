package hub

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Timeout struct {
	mutex    *sync.Mutex
	Active   bool
	stop     chan struct{}
	Callback func()
}

func NewTimeout(cb func()) *Timeout {
	return &Timeout{
		mutex: &sync.Mutex{},
		Active: false,
		stop: make(chan struct{}),
		Callback: cb,
	}
}

func (t *Timeout) Start() {
	t.mutex.Lock()
	if !t.Active {
		return
	}
	t.Active = true
	t.mutex.Unlock()

	for {
		timer := time.NewTimer(10 * time.Minute)
		select {
		case <-t.stop:
			timer.Stop()
			t.mutex.Lock()
			t.Active = false
			t.mutex.Unlock()
			return
		case <-timer.C:
			logrus.Infoln("- RESSOURCE - timeout after 10min -> quit")
			t.Callback()
		}
	}

}

func (t *Timeout) Stop() {
	if t.Active {
		t.stop <- struct{}{}
	}
}
