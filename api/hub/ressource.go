package hub

import (
	"fmt"
	"sync"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/stream"
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
		r.Data = container.Streams.Logs.Get()
		fmt.Println("[RESSOURCE] logs.recv created", r.Data)
	case "metrics":
		r.Data = container.Streams.Metrics.Get()
		fmt.Println("[RESSOURCE] metrics.recv created", r.Data)
	default:
		fmt.Println("[RESSOURCE] ressource init unkown event type:", r.Event)
		// remove failed ressource
		return
	}
	if r.Data == nil {
		fmt.Printf("[RESSOURCE] failed create %s receiver\n", r.Event)
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
	clientN := len(r.Receivers)
	r.mutex.Lock()
	idx := r.ClientIdx(c)
	if idx != -1 {
		r.Receivers[idx] = &Client{}

		// remove receiver
		r.Receivers = append(r.Receivers[:idx], r.Receivers[idx+1:]...)

		fmt.Printf("[RESSOURCE] client remove bef: %d aft: %d\n", clientN, len(r.Receivers))
	} else {
		fmt.Printf("[RESSOURCE] client remove: client not found %d\n", idx)
	}
	r.mutex.Unlock()
}

func (r *Ressource) Quit() {
	fmt.Println("[RESSOURCE] quit")
	r.Data.Quit()
}
