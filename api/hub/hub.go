package hub

import (
	"errors"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/events"
	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type Hub struct {
	mutex    *sync.RWMutex
	Sub      chan *Demand
	USub     chan *Demand
	Lve      chan *Client
	ResLeave chan *Ressource

	Ctr        *controller.Controller
	Ressources map[*container.Container][]*Ressource
}

func NewHub(ctr *controller.Controller) *Hub {
	return &Hub{
		mutex:      &sync.RWMutex{},
		Ctr:        ctr,
		Ressources: make(map[*container.Container][]*Ressource),
		Sub:        make(chan *Demand),
		USub:       make(chan *Demand),
		Lve:        make(chan *Client),
		ResLeave:   make(chan *Ressource),
	}
}

func (h *Hub) CreateClient(con *websocket.Conn) *Client {
	return NewClient(con, h.Sub, h.USub, h.Lve)
}

func (h *Hub) RemoveRessource(c *container.Container, r *Ressource) {
	logrus.Infof("- HUB - removed ressource %s for %s\n", r.Event, c.ID)
	if len(r.Subscribers) == 0 {
		ressources := h.Ressources[c]
		rIdx := slices.Index(ressources, r)
		if rIdx > -1 {
			h.mutex.Lock()

			// zero out + remove ressource
			h.Ressources[c][rIdx] = &Ressource{}
			h.Ressources[c] = append(ressources[:rIdx], ressources[rIdx+1:]...)
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) Subscribe(dem *Demand) {
	logrus.Infoln("- HUB - subscribe")
	h.mutex.Lock()
	res, exists := h.Ressource(dem.CID, dem.Ressource)
	if !exists {
		fmt.Println("res sub: res does not exist create new one")
		err := h.CreateRessource(dem.CID, dem.Ressource)
		if err != nil {
			logrus.Errorf("- HUB - failed to create ressource: %s\n", err)
			dem.Client.Error(err.Error())
			return
		}
		fresh, _ := h.Ressource(dem.CID, dem.Ressource)
		h.mutex.Unlock()
		fresh.Add <- dem.Client
		return
	}
	h.mutex.Unlock()
	res.Add <- dem.Client
}

func (h *Hub) Unsubscribe(dem *Demand) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	res, exists := h.Ressource(dem.CID, dem.Ressource)
	if !exists {
		logrus.Errorln("- HUB - failed to unsubscribe ressource: doesnt exist")
		// send client error message
		return
	}
	res.Rm <- dem.Client
}

func (h *Hub) BroadcastEvent(e events.Message) {
	logrus.Infoln("- HUB - broadcasting event message to all clients")

	// clients that already received that event
	var received map[*Client]bool

	eventMsg := map[string]string{
		"event_type": e.Action,
	}
	event := &Response{
		CID:     e.ID,
		Type:    "container_event",
		Message: eventMsg,
	}

	h.mutex.Lock()
	for _, ressources := range h.Ressources {
		for i := range ressources {
			for receiver := range ressources[i].Subscribers {
				if _, exists := received[receiver]; !exists {
					receiver.In <- event
				}
			}
		}
	}
	h.mutex.Unlock()

}

func (h *Hub) CreateRessource(cid, typ string) (err error) {
	container, exists := h.Ctr.Container(cid)
	if !exists {
		return errors.New("container doesnt exist")
	}

	r := NewRessource(container, typ, h.ResLeave)
	h.Ressources[container] = append(h.Ressources[container], r)
	err = r.Init()
	if err != nil {
		return
	}
	go r.Handle()

	return
}

func (h *Hub) Ressource(cid string, event string) (r *Ressource, exists bool) {
	container, exists := h.Ctr.Container(cid)
	if !exists {
		fmt.Println("res: container doesnt exist")
		return r, false
	}
	ressources, found := h.Ressources[container]
	if found {
		for i := range ressources {
			if ressources[i].container.ID == cid && ressources[i].Event == event {
				return ressources[i], true
			}
		}
	}
	return r, false
}

func (h *Hub) ClientLeave(c *Client) {
	fmt.Println("client leave")
	for _, ressources := range h.Ressources {
		for _, res := range ressources {
			res.Rm <- c
		}
	}
}

func (h *Hub) RessourceLeave(res *Ressource) {
	container, exists := h.Ctr.Container(res.container.ID)
	if exists {
		h.RemoveRessource(container, res)
	}
	logrus.Infoln("- HUB - ressource removed")
}

func (h *Hub) Run() {
	logrus.Infoln("- HUB - running")
	for {
		select {
		case dem := <-h.Sub:
			h.Subscribe(dem)
		case dem := <-h.USub:
			h.Unsubscribe(dem)
		case client := <-h.Lve:
			fmt.Println("hub rcvd client leave")
			h.ClientLeave(client)
		case res := <-h.ResLeave:
			h.RessourceLeave(res)
		}
	}

}
