package controller

import (
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	c       *client.Client
	Storage *Storage
	Events  *Events
}

func NewController() (ctr *Controller, err error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Controller{
		c:       c,
		Storage: NewStorage(c),
	}, err
}

func (ctr *Controller) Init() error {
	rawTypes, err := ctr.Storage.Discover()
	if err != nil {
		return err
	}
	err = ctr.Storage.Add(rawTypes...)
	if err != nil {
		return err
	}
	go ctr.Storage.Links()

	ctr.Events = NewEvents(ctr.c, ctr.Storage)
	go ctr.Events.Run()

	err = ctr.Storage.DB.Init()

	logrus.Infoln("- CONTROLLER - started")
	return err
}

func (ctr *Controller) Quit() {
	// complete this
	ctr.c.Close()
	logrus.Infoln("- CONTROLLER - quit")
}

func (ctr *Controller) Container(id string) (container *container.Container, found bool) {
	return ctr.Storage.Container(id)
}
