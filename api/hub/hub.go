package hub

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"golang.org/x/exp/slices"
)

type Hub struct {
	mutex      *sync.RWMutex
	Eps        *Endpoints
	Ctr        *controller.Controller
	Ressources map[*container.Container][]*Ressource
}

type Endpoints struct {
	Subscribe   chan *Demand
	Unsubscribe chan *Demand
	Leave       chan *Client
	Relay       chan *stream.Set
}

func NewHub(ctr *controller.Controller) *Hub {
	return &Hub{
		mutex:      &sync.RWMutex{},
		Ctr:        ctr,
		Ressources: make(map[*container.Container][]*Ressource),
		Eps: &Endpoints{
			Subscribe:   make(chan *Demand),
			Unsubscribe: make(chan *Demand),
			Leave:       make(chan *Client),
			Relay:       make(chan *stream.Set),
		},
	}
}

func (h *Hub) CreateClient(con *websocket.Conn) *Client {
	return NewClient(con, h.Eps)
}

func (h *Hub) HandleRessource(container *container.Container, r *Ressource) {
	fmt.Printf("[HUB] ressource init event:%s for %s\n", r.Event, r.ContainerID)
	var sub stream.Subscriber
	var err error

	switch r.Event {
	case "logs":
		sub, err = container.Streams.Logs.Subscribe()
	case "metrics":
		sub, err = container.Streams.Metrics.Subscribe()
	default:
		fmt.Println("cannot handle unkown ressource", r.Event)
		return
	}

	if err != nil {
		fmt.Println("failed to connect ressource with stream", err)
	}

	in := make(chan *stream.Set)
	go sub.Handle(in)
	defer func() {
		fmt.Println("defer: quit")
		sub.Quit()
		h.RemoveRessource(container, r)
	}()
	fmt.Println("[HUB] sub is handling stream")

	for data := range in {
		fmt.Println("handling incoming data", data)
		frame := &ResponseFrame{
			CID:     r.ContainerID,
			Type:    r.Event,
			Content: data.Data,
		}
		r.mutex.RLock()
		for i := range r.Receivers {
			fmt.Printf("supplying to %d receivers\n", len(r.Receivers))
			if len(r.Receivers) == -1 {
				fmt.Println("everyone left quiting ressource")
				return
			}
			r.Receivers[i].In <- frame
		}
		r.mutex.RUnlock()
	}
}

func (h *Hub) RemoveRessource(c *container.Container, r *Ressource) {
	fmt.Println("removing ressource")
	if len(r.Receivers) == 0 {
		ressources := h.Ressources[c]
		rIdx := slices.Index(ressources, r)
		if rIdx > -1 {
			h.mutex.Lock()
			h.Ressources[c][rIdx] = &Ressource{}
			slices.Delete(h.Ressources[c], rIdx, rIdx)
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) Subscribe(dem *Demand) {
	fmt.Printf("[HUB::subscribe] %s %s\n", dem.Ressource, dem.CID)
	container, exists := h.Ctr.ContainerGet(dem.CID)
	if !exists {
		fmt.Println("[HUB] container not found")
		return
	}

	res, exists := h.Ressource(dem.CID, dem.Ressource)
	if !exists {
		fmt.Println("[HUB] creating new ressource")
		// create new ressource
		r := NewRessource(dem.CID, dem.Ressource, []*Client{dem.Client})
		fmt.Println("new res nil?", r == nil)
		h.mutex.Lock()
		h.Ressources[container] = []*Ressource{r}
		go h.HandleRessource(container, r)
		h.mutex.Unlock()

	} else {
		fmt.Println("[HUB] ressource found, adding client")
		res.mutex.Lock()
		res.Receivers = append(res.Receivers, dem.Client)
		res.mutex.Unlock()
	}
}

func (h *Hub) Unsubscribe(dem *Demand) {
	fmt.Printf("[HUB::unsubscribe] %s %s\n", dem.Ressource, dem.CID)
	_, exists := h.Ctr.ContainerGet(dem.CID)
	if !exists {
		return
	}

	res, exists := h.Ressource(dem.CID, dem.Ressource)
	if exists {
		for i := range res.Receivers {
			if res.Receivers[i] == dem.Client {
				res.RemoveClient(dem.Client)
				return
			}

		}
	}

}

func (h *Hub) Ressource(cid string, event string) (r *Ressource, exists bool) {
	container, exists := h.Ctr.ContainerGet(cid)
	if !exists {
		return r, false
	}
	ressources, found := h.Ressources[container]
	if found {
		for i := range ressources {
			if ressources[i].ContainerID == cid && ressources[i].Event == event {
				return ressources[i], true
			}
		}
	}
	return r, false
}

func (h *Hub) ClientLeave(c *Client) {
	for _, ressources := range h.Ressources {
		for _, res := range ressources {
			fmt.Println("deleting client from ressource")
			res.mutex.Lock()
			res.RemoveClient(c)
			res.mutex.Unlock()
		}
	}
}

func (h *Hub) Run() {
	fmt.Println("hub running...")

	for {
		select {
		case dem := <-h.Eps.Subscribe:
			h.Subscribe(dem)
		case dem := <-h.Eps.Unsubscribe:
			h.Unsubscribe(dem)
		case client := <-h.Eps.Leave:
			h.ClientLeave(client)
		}
	}
}
