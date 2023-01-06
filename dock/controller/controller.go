package controller

import (
	"github.com/docker/docker/client"
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