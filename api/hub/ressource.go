package hub

import (
	"sync"

	"github.com/h0rzn/monitoring_agent/dock/stream"
	"golang.org/x/exp/slices"
)

type Ressource struct {
	mutex       *sync.RWMutex
	ContainerID string
	Event       string
	Data        stream.Subscriber
	Receivers   []*Client
}

func NewRessource(cid string, event string, receivers []*Client) *Ressource {
	return &Ressource{
		mutex:       &sync.RWMutex{},
		ContainerID: cid,
		Event:       event,
		Data:        nil,
		Receivers:   receivers,
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
	if idx > -1 {
		r.Receivers[idx] = &Client{}
		slices.Delete(r.Receivers, idx, idx)
	}
	r.mutex.Unlock()
}
