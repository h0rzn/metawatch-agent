package controller

import (
	"fmt"

	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
)

type Controller struct {
	c       *client.Client
	Storage *Storage
}

func NewController() (ctrl *Controller, err error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Controller{
		c:       c,
		Storage: NewStorage(c),
	}, err
}

func (ctrl *Controller) Init() error {
	rawTypes, err := ctrl.Storage.Discover()
	if err != nil {
		fmt.Println("discover err:", err)
		return err
	}
	err = ctrl.Storage.AddAll(rawTypes)
	return err
}

func (ctrl *Controller) Quit() {
	// complete this
	ctrl.c.Close()
}

func (ctrl *Controller) Container(id string) *container.Container {
	return ctrl.Storage.Container(id)
}

func (ctrl *Controller) ContainerGet(id string) (container *container.Container, found bool) {
	for i := range ctrl.Storage.Containers {
		if ctrl.Storage.Containers[i].ID == id {
			return ctrl.Storage.Containers[i], true
		}
	}
	return container, false
}
