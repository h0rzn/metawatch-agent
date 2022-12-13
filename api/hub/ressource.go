package hub

import (
	"fmt"
	"sync"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

type Ressource struct {
	mutex       *sync.RWMutex
	ContainerID string
	Event       string
	DataRcv     *stream.Receiver
	Add         chan *Client
	Rm          chan *Client
	Receivers   map[*Client]bool
}

func NewRessource(cid string, event string) *Ressource {
	return &Ressource{
		mutex:       &sync.RWMutex{},
		ContainerID: cid,
		Event:       event,
		Add:         make(chan *Client),
		Rm:          make(chan *Client),
		Receivers:   make(map[*Client]bool),
	}
}

func (r *Ressource) SetStreamer(container *container.Container) {
	var rcv *stream.Receiver
	var err error

	switch r.Event {
	case "logs":
		rcv, err = container.Streams.Logs.Get(false)
		logrus.Infoln("- RESSOURCE - logs.recv created\n")
	case "metrics":
		rcv, err = container.Streams.Metrics.Get(false)
		logrus.Infoln("- RESSOURCE - metrics.recv created\n")
	default:
		logrus.Errorf("- RESSOURCE - ressource init unkown event type: %s\n", r.Event)
		// remove failed ressource
		return
	}
	if err != nil {
		logrus.Errorf("- RESSOURCE - failed create %s receiver: %s\n", r.Event, err)
	}
	r.DataRcv = rcv
}

func (r *Ressource) Handle() <-chan struct{} {
	closed := make(chan struct{})
	go func() {
		for {
			select {
			case <-r.DataRcv.Closing:
				fmt.Println("ressource: received closing sig from receiver")
				r.Quit()
				closed <- struct{}{}
				return
			case client := <-r.Add:
				r.addClient(client)
			case client := <-r.Rm:
				r.rmClient(client)
			case set, ok := <-r.DataRcv.In:
				if !ok {
					fmt.Println("ressource: data_rcv in closed")
					return
				}
				r.broadcast(set)
			}
		}
	}()
	return closed
}

func (r *Ressource) broadcast(set stream.Set) {
	r.mutex.Lock()
	frame := &ResponseFrame{
		CID:     r.ContainerID,
		Type:    r.Event,
		Content: set.Data,
	}
	for client := range r.Receivers {
		client.In <- frame
	}
	r.mutex.Unlock()
}

func (r *Ressource) addClient(c *Client) {
	logrus.Debugln("- RESSOURCE - adding client")
	r.mutex.Lock()
	r.Receivers[c] = true
	c.Handle()
	r.mutex.Unlock()
}

func (r *Ressource) rmClient(c *Client) {
	r.mutex.Lock()
	err := c.Close()
	if err != nil {
		logrus.Errorf("- RESSOURCE - client close err: %s\n", err)
	}
	delete(r.Receivers, c)
	r.mutex.Unlock()
}

func (r *Ressource) Quit() {
	logrus.Info("- RESSOURCE - quit")
	for client := range r.Receivers {
		fmt.Println("ressource quit: kill client")
		r.Rm <- client
	}
}
