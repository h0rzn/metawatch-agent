package hub

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
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
	logrus.Infoln("- HUB - handling ressource")
	<-r.Handle()
	h.RemoveRessource(container, r)
}

func (h *Hub) RemoveRessource(c *container.Container, r *Ressource) {
	logrus.Infof("- HUB - removed ressource %s for %s\n", r.Event, c.ID)
	if len(r.Receivers) == 0 {
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
	logrus.Infof("- HUB - subscribing: res:%s cid:%s\n", dem.Ressource, dem.CID)
	_, exists := h.Ctr.Container(dem.CID)
	if !exists {
		logrus.Errorf("- HUB - container %s not found\n", dem.CID)
		return
	}

	h.mutex.Lock()
	res, exists := h.Ressource(dem.CID, dem.Ressource)
	if !exists {
		logrus.Errorf("- HUB - ressource for %s not found\n", res.ContainerID)
		return
	}
	res.Add <- dem.Client
	h.mutex.Unlock()
}

func (h *Hub) Unsubscribe(dem *Demand) {
	logrus.Infof("- HUB - unsubscribe res:%s cid:%s\n", dem.Ressource, dem.CID)
	_, exists := h.Ctr.Container(dem.CID)
	if !exists {
		return
	}

	res, exists := h.Ressource(dem.CID, dem.Ressource)
	if exists {
		res.Rm <- dem.Client
	}

}

func (h *Hub) CreateRessource(cid, typ string) {
	r := NewRessource(cid, typ)
	container, exists := h.Ctr.Container(cid)
	if !exists {
		logrus.Errorf("- HUB - container %s not found\n", cid)
		return
	}
	h.Ressources[container] = append(h.Ressources[container], r)
	r.SetStreamer(container)
	go h.HandleRessource(container, r)
}

func (h *Hub) Ressource(cid string, event string) (r *Ressource, exists bool) {
	container, exists := h.Ctr.Container(cid)
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
			res.Rm <- c
		}
	}
}

func (h *Hub) Run() {
	logrus.Infoln("- HUB - running")

	for {
		select {
		case dem := <-h.Eps.Subscribe:
			if _, exists := h.Ressource(dem.CID, dem.Ressource); !exists {
				h.CreateRessource(dem.CID, dem.Ressource)
			}
			h.Subscribe(dem)
		case dem := <-h.Eps.Unsubscribe:
			h.Unsubscribe(dem)
		case client := <-h.Eps.Leave:
			h.ClientLeave(client)
		}
	}
}
