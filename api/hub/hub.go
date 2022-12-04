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
	logrus.Infof("- HUB - handling ressource receiver for %d receivers\n", len(r.Receivers))

	r.SetStreamer(container)
	for {
		set, more := <-r.Data.In
		if more {
			logrus.Debugf("- HUB - handling ressource set for %d receivers\n", len(r.Receivers))
			frame := &ResponseFrame{
				CID:     r.ContainerID,
				Type:    r.Event,
				Content: set.Data,
			}
			h.mutex.RLock()
			for idx := range r.Receivers {
				r.Receivers[idx].In <- frame
			}
			h.mutex.RUnlock()

			if len(r.Receivers) == 0 {
				break
			}
		} else {
			r.Quit()
			h.RemoveRessource(container, r)
		}
	}
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
	container, exists := h.Ctr.Container(dem.CID)
	if !exists {
		logrus.Errorf("- HUB - container %s not found\n", dem.CID)
		return
	}

	res, exists := h.Ressource(dem.CID, dem.Ressource)
	h.mutex.Lock()
	if !exists {
		logrus.Info("[HUB] creating new ressource")
		// create new ressource
		r := NewRessource(dem.CID, dem.Ressource, dem.Client)
		h.Ressources[container] = append(h.Ressources[container], r)
		go h.HandleRessource(container, r)
	} else {
		logrus.Info("[HUB] ressource found, adding client")
		res.Receivers = append(res.Receivers, dem.Client)
	}
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
		for i := range res.Receivers {
			if res.Receivers[i] == dem.Client {
				res.RemoveClient(dem.Client)
				// quit receiver
				return
			}

		}
	}

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
			res.RemoveClient(c)
			logrus.Infof("- HUB - removed client: %d left\n", len(res.Receivers))
		}
	}
}

func (h *Hub) Run() {
	logrus.Infoln("- HUB - running")

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
