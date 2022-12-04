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
	Data        *stream.Receiver
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
	switch r.Event {
	case "logs":
		r.Data = container.Streams.Logs.Get(false)
		logrus.Infof("- RESSOURCE - logs.recv created\n", r.Data)
	case "metrics":
		r.Data = container.Streams.Metrics.Get(false)
		logrus.Infof("- RESSOURCE - metrics.recv created\n", r.Data)
	default:
		logrus.Errorf("- RESSOURCE - ressource init unkown event type: %s\n", r.Event)
		// remove failed ressource
		return
	}
	if r.Data == nil {
		logrus.Errorf("- RESSOURCE - failed create %s receiver\n", r.Event)
	}
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
		r.Receivers[idx] = &Client{}

		// remove receiver
		r.Receivers = append(r.Receivers[:idx], r.Receivers[idx+1:]...)
		logrus.Debugln("- RESSOURCE - client removed\n")

	} else {
		logrus.Warnln("- RESSOURCE-   client remove: client not found %d\n", idx)
	}
	r.mutex.Unlock()
}

func (r *Ressource) Quit() {
	logrus.Info("- RESSOURCE - quit")
	r.Data.Quit()
}
