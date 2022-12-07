package hub

import (
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
	Receivers   []*Client
}

func NewRessource(cid string, event string, receiver *Client) *Ressource {
	receivers := []*Client{receiver}
	return &Ressource{
		mutex:       &sync.RWMutex{},
		ContainerID: cid,
		Event:       event,
		Receivers:   receivers,
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

func (r *Ressource) ClientIdx(c *Client) int {
	for i := range r.Receivers {
		if r.Receivers[i] == c {
			return i
		}
	}
	return -1
}

func (r *Ressource) RemoveClient(c *Client) {
	r.mutex.Lock()
	idx := r.ClientIdx(c)
	if idx != -1 {
		c.Done <- struct{}{}
		r.Receivers[idx] = &Client{}

		// remove receiver
		r.Receivers = append(r.Receivers[:idx], r.Receivers[idx+1:]...)
		logrus.Debugln("- RESSOURCE - client removed\n")

	} else {
		logrus.Warnln("- RESSOURCE -   client remove: client not found %d\n", idx)
	}
	r.mutex.Unlock()
}

func (r *Ressource) Quit() {
	logrus.Info("- RESSOURCE - quit")
	r.mutex.Lock()
	for _, client := range r.Receivers {
		client.ForceLeave()
	}
	r.mutex.Unlock()
}
