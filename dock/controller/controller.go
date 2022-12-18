package controller

import (
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/image"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	c       *client.Client
	Storage *Storage
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
	logrus.Infoln("- CONTROLLER - starting")
	err := ctr.Storage.Init()
	return err
}

func (ctr *Controller) Quit() {
	// complete this
	ctr.c.Close()
	logrus.Infoln("- CONTROLLER - quit")
}

func (ctr *Controller) Container(id string) (container *container.Container, exists bool) {
	return ctr.Storage.ContainerStore.Container(id)
}

func (ctr *Controller) Image(id string) (image *image.Image, exists bool) {
	return ctr.Storage.ImageStore.Image(id)
}
