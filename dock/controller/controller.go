package controller

import (
	"context"
	"fmt"

	dock_events "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/events"
	"github.com/h0rzn/monitoring_agent/dock/image"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	c *client.Client
	// Storage    *Storage
	DB         *db.DB
	About      *About
	Events     *events.Events
	Containers *container.Storage
	Images     *image.Storage
}

type About struct {
	Version    string `json:"version"`
	APIVersion string `json:"api_version"`
	OS         string `json:"os"`
	ImageN     int    `json:"image_n"`
	ContainerN int    `json:"container_n"`
	// Plugins?
}

func (i *About) Update(c *client.Client) (err error) {
	ctx := context.Background()
	version, err := c.ServerVersion(ctx)
	if err != nil {
		return
	}
	i.Version = version.Version
	i.APIVersion = version.APIVersion
	i.OS = version.Os

	ctx = context.Background()
	info, err := c.Info(ctx)
	if err != nil {
		return
	}
	i.ImageN = info.Images
	i.ContainerN = info.Containers

	return
}

func NewController() (ctr *Controller, err error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Controller{
		c:          c,
		DB:         &db.DB{},
		About:      &About{},
		Events:     events.NewEvents(c),
		Containers: container.NewStorage(c),
		Images:     image.NewStorage(c),
	}, err
}

func (ctr *Controller) Init() (err error) {
	logrus.Infoln("- CONTROLLER - starting")
	err = ctr.About.Update(ctr.c)
	if err != nil {
		logrus.Warnf("- CONTROLLER - about might not be complete, err: %s\n", err)
	}

	err = ctr.Events.Init()
	if err != nil {
		return err
	}
	go ctr.HandleEvents()

	err = ctr.Images.Init()
	if err != nil {
		logrus.Errorf("- STORAGE - (images) failed to init: %s\n", err)
		return
	}

	err = ctr.Containers.Init(ctr.Images.ByID)
	if err != nil {
		logrus.Errorf("- STORAGE - (containers) failed to init: %s\n", err)
		return
	}
	go func() {
		for items := range ctr.Containers.Broadcast() {
			go ctr.DB.Client.BulkWrite(items)
		}
		fmt.Println("feed writer left")
	}()

	err = ctr.DB.Init()
	if err != nil {
		logrus.Errorf("- STORAGE - (db) failed to init: %s\n", err)
	}
	return err
}

func (ctr *Controller) HandleEvents() {
	eventRcv, err := ctr.Events.Get()
	if err != nil {
		logrus.Errorf("- CONTROLLER - failed to run event handler: %s\n", err)
		return
	}

	logrus.Infoln("- CONTROLLER - running event handler...")
	for set := range eventRcv.In {
		fmt.Println("handling event")
		event := set.Data.(dock_events.Message)
		// add queue
		if event.Type != dock_events.ContainerEventType {
			continue
		}
		switch event.Status {
		case "start":
			ctr.ContainerStart(event)
		case "stop":
			ctr.ContainerStop(event)
		case "destroy":
			ctr.ContainerDestroy(event)
		default:
			logrus.Warnf("- CONTROLLER - event %s is unkown or not implemented\n", event.Status)
		}
		ctr.About.Update(ctr.c)
	}
}

func (ctr *Controller) ContainerStart(e dock_events.Message) {
	err := ctr.Containers.Add(e.ID)
	logEventExec(err, e)
}

func (ctr *Controller) ContainerStop(e dock_events.Message) {
	err := ctr.Containers.Stop(e.ID)
	logEventExec(err, e)
}

func (ctr *Controller) ContainerDestroy(e dock_events.Message) {
	err := ctr.Containers.Remove(e.ID)
	logEventExec(err, e)
}

func (ctr *Controller) Quit() {
	// complete this
	ctr.c.Close()
	logrus.Infoln("- CONTROLLER - quit")
}

func logEventExec(err error, e dock_events.Message) {
	if err != nil {
		logrus.Errorf("- CONTROLLER - exec of event %s failed: %s\n", e.Status, err)
	} else {
		logrus.Infof("- CONTROLLER - exec of event %s successful\n", e.Status)
	}
}
