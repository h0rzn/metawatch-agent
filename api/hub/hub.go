package hub

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/dock/controller"
	"github.com/sirupsen/logrus"
)

type Hub struct {
	mutex     *sync.RWMutex
	Sub       chan *Demand
	USub      chan *Demand
	Lve       chan *Client
	Ctr       *controller.Controller
	Resources map[Resource]bool
	LveSig    chan Resource
}

func NewHub(ctr *controller.Controller) *Hub {
	return &Hub{
		mutex:     &sync.RWMutex{},
		Ctr:       ctr,
		Resources: make(map[Resource]bool),
		Sub:       make(chan *Demand),
		USub:      make(chan *Demand),
		Lve:       make(chan *Client),
		LveSig:    make(chan Resource),
	}
}

func (h *Hub) CreateClient(con *websocket.Conn) *Client {
	return NewClient(con, h.Sub, h.USub, h.Lve)
}

func (h *Hub) CreateGeneric(cid, typ string) (*GenericR, error) {
	logrus.Debugln("- HUB - creating generic resource")
	if container, exists := h.Ctr.Containers.Container(cid); exists {
		r := NewGenericR(typ, container, h.LveSig)
		err := r.Run()
		if err != nil {
			return &GenericR{}, err
		}
		h.Resources[r] = true
		return r, err
	}
	return &GenericR{}, fmt.Errorf("cannot find container %s", cid)
}

func (h *Hub) CreateCombined(cid, typ string) (*CombindedMetrics, error) {
	logrus.Debugln("- HUB - creating combined resource")
	if cid != "_all" {
		return &CombindedMetrics{}, fmt.Errorf("failed to create combined resource: cid as to be '_all' got: %s", cid)
	}

	r := NewCombinedR(h.Ctr.Containers, h.LveSig)
	err := r.Run()
	if err != nil {
		return &CombindedMetrics{}, err
	}
	h.Resources[r] = true
	return r, err
}

func (h *Hub) CreateEvents() (*EventsR, error) {
	logrus.Debugln("- HUB - creating events resource")
	r, exists := h.hasEventR()
	if exists {
		return r, nil
	}

	r = NewEventsR(h.Ctr.Events.Get, h.LveSig)
	err := r.Run()
	if err != nil {
		return &EventsR{}, err
	}
	h.Resources[r] = true

	return r, nil
}

func (h *Hub) hasEventR() (*EventsR, bool) {
	for r := range h.Resources {
		if r.Type() == "event" {
			return (r).(*EventsR), true
		}
	}
	return nil, false
}

func (h *Hub) Subscribe(dem *Demand) {
	logrus.Infoln("- HUB - Subscribe")
	h.mutex.Lock()
	if res, exists := h.Resource(dem.CID, dem.Ressource); exists {
		fmt.Println("resource exists: adding client")
		res.Add(dem.Client)
	} else {
		switch dem.Ressource {
		case "metrics", "logs":
			r, err := h.CreateGeneric(dem.CID, dem.Ressource)
			if err != nil {
				logrus.Errorf("resource creation err: %s\n", err)
				dem.Client.Error(err.Error())
				break
			}
			r.Add(dem.Client)
		case "combined_metrics":
			r, err := h.CreateCombined(dem.CID, dem.Ressource)
			if err != nil {
				logrus.Errorf("resource creation err: %s\n", err)
				dem.Client.Error(err.Error())
				break
			}
			r.Add(dem.Client)
		case "events":
			r, err := h.CreateEvents()
			if err != nil {
				logrus.Errorf("resource creation err: %s\n", err)
				dem.Client.Error(err.Error())
				break
			}
			r.Add(dem.Client)
		default:
			dem.Client.Error(fmt.Sprintf("cannot create resource, container %s or type %s does not exist", dem.CID, dem.Ressource))
		}
	}
	h.mutex.Unlock()

}

func (h *Hub) Unsubscribe(dem *Demand) {
	logrus.Infoln("- HUB - Unsubscribe")
	h.mutex.Lock()
	if r, exists := h.Resource(dem.CID, dem.Ressource); exists {
		r.Rm(dem.Client)
	} else {
		logrus.Errorf("- HUB - failed to unsubscribe: resource not found")
		dem.Client.Error("failed to unsubscribe, resource not found")
	}
	h.mutex.Unlock()
}

func (h *Hub) Remove(r Resource) {
	r.Quit()
	h.mutex.Lock()
	delete(h.Resources, r)
	h.mutex.Unlock()
}

func (h *Hub) Resource(cid string, event string) (r Resource, exists bool) {
	logrus.Debugf("resource get: cid %s, typ %s\n", cid, event)
	for res := range h.Resources {
		if res.CID() == cid && res.Type() == event {
			fmt.Println("resource get: found resource")
			return res, true
		}
	}
	return
}

func (h *Hub) ClientLeave(c *Client) {
	fmt.Println("client leave")
	for r := range h.Resources {
		r.Rm(c)
	}
}

func (h *Hub) RessourceLeave(res Resource) {
	h.mutex.Lock()
	delete(h.Resources, res)
	h.mutex.Unlock()
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
		case res := <-h.LveSig:
			h.RessourceLeave(res)
		}
	}

}
