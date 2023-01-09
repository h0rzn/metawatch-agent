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

func (h *Hub) CreateGeneric(cid, typ string) error {
	fmt.Println("creating generic")
	if container, exists := h.Ctr.Containers.Container(cid); exists {
		fmt.Println("container exists")
		r := NewGenericR(typ, container, h.LveSig)
		fmt.Println("new generic created")
		h.Resources[r] = true
		err := r.Run()
		if err != nil {
			return err
		}
		fmt.Println("new generic running")
		return nil
	}
	return fmt.Errorf("cannot find container %s", cid)
}

func (h *Hub) CreateEvents() error {
	if !h.hasEventR() {
		r := NewEventsR(h.Ctr.Events.Get, h.LveSig)
		h.Resources[r] = true
		err := r.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Hub) hasEventR() bool {
	for r := range h.Resources {
		if r.Type() == "event" {
			return true
		}
	}
	return false
}

func (h *Hub) Subscribe(dem *Demand) {
	logrus.Infoln("- HUB - Subscribe")
	h.mutex.Lock()
	if res, exists := h.Resource(dem.CID, dem.Ressource); exists {
		fmt.Println("resource exists: adding client")
		res.Add(dem.Client)
	} else {
		fmt.Println("resource does not exist: create resource + adding client")
		// create new resource
		switch dem.Ressource {
		case "metrics", "logs":
			err := h.CreateGeneric(dem.CID, dem.Ressource)
			if err != nil {
				dem.Client.Error(err.Error())
				break
			}
			fmt.Println("attempt to add client to generic")
			if r, exists := h.Resource(dem.CID, dem.Ressource); exists {
				fmt.Println("recheck: resource exists")
				r.Add(dem.Client)
			}
		case "events":
			fmt.Println("attempt to create eventsr")
			err := h.CreateEvents()
			if err != nil {
				fmt.Println("eventsr create err", err)
				dem.Client.Error(err.Error())
				break
			}
			fmt.Println("create events err", err)
			if r, exists := h.Resource(dem.CID, dem.Ressource); exists {
				fmt.Println("eventsr exists add client")
				r.Add(dem.Client)
				fmt.Println("eventsr client added")
			}
			fmt.Println("events doesnt exist")
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
	for res := range h.Resources {
		fmt.Println("trying to identify")
		if res.CID() == cid && res.Type() == event {
			fmt.Println("resource found")
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
