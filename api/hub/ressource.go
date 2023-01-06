package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

const RessourceTimeout = 10 * time.Minute

type Ressource struct {
	mutex       *sync.RWMutex
	Event       string
	container   *container.Container
	Receiver    *stream.Receiver
	Add         chan *Client
	Rm          chan *Client
	LveSig      chan *Ressource
	done        chan struct{}
	Subscribers map[*Client]bool
	TOActive    bool
	TOFired     chan struct{}
	TOStop      chan struct{}
}

func NewRessource(container *container.Container, event string, lveSig chan *Ressource) *Ressource {
	return &Ressource{
		mutex:       &sync.RWMutex{},
		container:   container,
		Event:       event,
		Add:         make(chan *Client),
		Rm:          make(chan *Client),
		LveSig:      lveSig,
		done:        make(chan struct{}),
		Subscribers: make(map[*Client]bool),
		TOActive:    false,
		TOFired:     make(chan struct{}),
		TOStop:      make(chan struct{}),
	}
}

func (r *Ressource) Init() error {
	logrus.Debugln("- RESSOURCE - init")
	var rcv *stream.Receiver
	var err error

	switch r.Event {
	case "logs":
		rcv, err = r.container.Streams.Logs.Get(false)
		logrus.Infoln("- RESSOURCE - logs.recv created\n")
	case "metrics":
		rcv, err = r.container.Streams.Metrics.Get(false)
		logrus.Infoln("- RESSOURCE - metrics.recv created\n")
	}

	r.Receiver = rcv
	return err
}

func (r *Ressource) Handle() {
	logrus.Debugln("- RESSOURCE - handling")
	for {
		select {
		case <-r.Receiver.Closing:
			fmt.Println("ressource: received closing sig from receiver")
			r.Quit()
			return
		case client := <-r.Add:
			r.addClient(client)
		case client := <-r.Rm:
			r.rmClient(client)
		case set, ok := <-r.Receiver.In:
			if !ok {
				fmt.Println("ressource: data_rcv in closed")
				return
			}
			r.broadcast(set)
		}
	}

}

func (r *Ressource) broadcast(set stream.Set) {
	r.mutex.Lock()
	frame := &Response{
		CID:     r.container.ID,
		Type:    r.Event,
		Message: set.Data,
	}
	for client := range r.Subscribers {
		client.In <- frame
	}
	r.mutex.Unlock()
}

func (r *Ressource) addClient(c *Client) {
	logrus.Debugln("- RESSOURCE - adding client")
	r.mutex.Lock()
	if r.TOActive {
		logrus.Infoln("- RESSOURCE - timeout stopeed (new client)")
		r.TOStop <- struct{}{}
	}
	r.Subscribers[c] = true
	r.mutex.Unlock()
}

func (r *Ressource) rmClient(c *Client) {
	r.mutex.Lock()
	err := c.Close()
	if err != nil {
		logrus.Errorf("- RESSOURCE - client close err: %s\n", err)
	}
	delete(r.Subscribers, c)

	if len(r.Subscribers) == 0 {
		if !r.TOActive {
			go r.Timeout()
		}
	}
	r.mutex.Unlock()
}

func (r *Ressource) Timeout() {
	r.mutex.Lock()
	r.TOActive = true
	r.mutex.Unlock()

	for {
		timer := time.NewTimer(RessourceTimeout)
		select {
		case <-r.TOStop:
			timer.Stop()
			r.mutex.Lock()
			r.TOActive = false
			r.mutex.Unlock()
			return
		case <-timer.C:
			logrus.Infoln("- RESSOURCE - timeout after 10min -> quit")
			r.Quit()
		}
	}
}

func (r *Ressource) Quit() {
	logrus.Infoln("- RESSOURCE - quit")
	for client := range r.Subscribers {
		r.rmClient(client)
	}
	r.LveSig <- r
}
