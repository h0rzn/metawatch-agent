package hub

import (
	"fmt"
	"sync"

	devents "github.com/docker/docker/api/types/events"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Resource interface {
	CID() string
	Type() string
	Run() error
	Add(*Client)
	Rm(*Client)
	Broadcast(stream.Set)
	Quit()
}

type GenericR struct {
	mutex     *sync.Mutex
	Typ       string
	Container *container.Container
	Input     *stream.Receiver
	Subs      map[*Client]bool
	LveSig    chan Resource
	Timeout *Timeout
}

func NewGenericR(typ string, cont *container.Container, lveSig chan Resource) *GenericR {
	r := &GenericR{
		mutex:     &sync.Mutex{},
		Typ:       typ,
		Container: cont,
		Subs:      make(map[*Client]bool),
		LveSig:    lveSig,
	}
	r.Timeout = NewTimeout(r.Quit)
	return r
}

func (r *GenericR) CID() string {
	return r.Container.ID
}

func (e *GenericR) Type() string {
	return e.Typ
}

func (r *GenericR) Run() error {
	switch r.Typ {
	case "logs":
		rcv, err := r.Container.Streams.Logs.Get(false)
		if err != nil {
			return err
		}
		r.Input = rcv
	case "metrics":
		rcv, err := r.Container.Streams.Metrics.Get(false)
		if err != nil {
			return err
		}
		r.Input = rcv
	default:
		return fmt.Errorf("unkown type: %s", r.Typ)
	}

	// handle resource
	go func() {
		for {
			select {
			case <-r.Input.Closing:
				r.Quit()
				return
			case set, ok := <-r.Input.In:
				if !ok {
					return
				}
				r.Broadcast(set)
			}
		}
	}()

	return nil
}

func (r *GenericR) Add(c *Client) {
	if len(c.Sub) == 0 {
		r.Timeout.Stop()
	}
	fmt.Println("generic: adding client")
	r.mutex.Lock()
	r.Subs[c] = true
	r.mutex.Unlock()
}

func (r *GenericR) Rm(c *Client) {
	r.mutex.Lock()
	delete(r.Subs, c)
	r.mutex.Unlock()
	if len(r.Subs) == 0 {
		r.Timeout.Start()
	}
}

func (r *GenericR) Broadcast(set stream.Set) {
	frame := &Response{
		CID:     r.Container.ID,
		Type:    set.Type,
		Message: set.Data,
	}
	r.mutex.Lock()
	for client := range r.Subs {
		client.In <- frame
	}
	r.mutex.Unlock()
}

func (r *GenericR) Quit() {
	r.mutex.Lock()

	for c := range r.Subs {
		r.Rm(c)
	}
	r.LveSig <- r
	r.mutex.Unlock()
}

type GetEvents func() (*stream.Receiver, error)

type EventsR struct {
	mutex  *sync.Mutex
	Typ    string
	Subs   map[*Client]bool
	LveSig chan Resource
	Events GetEvents
	Timeout *Timeout
}

func NewEventsR(getEvents GetEvents, lveSig chan Resource) *EventsR {
	r := &EventsR{
		mutex:  &sync.Mutex{},
		Typ:    "events",
		Subs:   make(map[*Client]bool),
		LveSig: lveSig,
		Events: getEvents,
	}
	r.Timeout = NewTimeout(r.Quit)
	return r
}

func (r *EventsR) CID() string {
	return ""
}

func (r *EventsR) Type() string {
	return r.Typ
}

func (r *EventsR) Run() error {
	fmt.Println("event resource running", r.Type())
	rcv, err := r.Events()
	if err != nil {
		return err
	}
	go func() {
		for set := range rcv.In {
			fmt.Println("handling event resource event")
			r.Broadcast(set)
		}
	}()

	return nil
}

func (r *EventsR) Add(c *Client) {
	if len(c.Sub) == 0 {
		r.Timeout.Stop()
	}
	fmt.Println("eventsr adding client")
	r.mutex.Lock()
	r.Subs[c] = true
	r.mutex.Unlock()
	fmt.Println("eventsr client added")
}

func (r *EventsR) Rm(c *Client) {
	r.mutex.Lock()
	delete(r.Subs, c)
	r.mutex.Unlock()
	if len(r.Subs) == 0 {
		r.Timeout.Start()
	}
}

func (r *EventsR) Broadcast(set stream.Set) {
	event, ok := set.Data.(devents.Message)
	if !ok {
		fmt.Println("type assert failed")
	}
	msg := &Response{
		Type: "event",
		Message: map[string]interface{}{
			"type": fmt.Sprintf("%s_%s", event.Type, event.Status),
			"id":   event.ID,
		},
	}
	fmt.Println("sending to clients, len", len(r.Subs))
	r.mutex.Lock()
	for c := range r.Subs {
		fmt.Println("event resource: sending to client")
		c.In <- msg
	}
	r.mutex.Unlock()
}

func (r *EventsR) Quit() {
	r.mutex.Lock()

	for c := range r.Subs {
		r.Rm(c)
	}
	r.LveSig <- r
	r.mutex.Unlock()
}
