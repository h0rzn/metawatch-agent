package controller

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Events struct {
	c      *client.Client
	Strg   *Storage
	Inform func(events.Message)
}

func NewEvents(c *client.Client, strg *Storage) *Events {
	return &Events{
		c:    c,
		Strg: strg,
	}
}

func (ev *Events) SetInformer(fn func(events.Message)) {
	ev.Inform = fn
}

func (ev *Events) Run() {
	logrus.Infoln("- EVENTS - running...")
	ctx := context.Background()
	evs, errs := ev.c.Events(ctx, types.EventsOptions{})
	for {
		select {
		case event := <-evs:
			ev.catch(event)
		case err := <-errs:
			logrus.Errorf("- EVENTS - erroring receiving events: %s\n", err.Error())
		}
	}

}

func (ev *Events) onStop(e events.Message) {
	err := ev.Strg.Remove(e.ID)
	if err != nil {
		logrus.Errorf("- EVENTS - failed to handle [stop] event: %s\n", err)
	} else {
		logrus.Infoln("- EVENTS - succesfully removed container based on [stop]")
	}
	ev.Inform(e)

}

func (ev *Events) onStart(e events.Message) {
	status := ev.Strg.Push(e.ID)
	switch status {
	case containerExists:
		logrus.Infoln("- EVENTS - ignoring [start]: container already indexed")
	case containerStartErr:
		logrus.Errorln("- EVENTS - failed to start container based on [start]")
	case containerAdded:
		logrus.Infoln("- EVENTS - succesfully added container based on [start]")
	}
	ev.Inform(e)
}

// https://docs.docker.com/engine/reference/commandline/events/
func (ev *Events) catch(event events.Message) {
	// ignore non container events
	if event.Type != events.ContainerEventType {
		return
	}

	// switch event.Status {
	// case "create":
	// case "destroy":
	// case "die":
	// case "health_status":
	// case "pause":
	// case "rename":
	// case "restart":
	// case "start":
	// case "stop":
	// case "unpause":
	// case "update":
	// 	break
	// default:
	// 	break
	// }

	switch event.Status {
	case "start":
		logrus.Infof("- EVENTS - handling container [%s] event %s", event.Status, event.From)
		ev.onStart(event)
	case "stop":
		logrus.Infof("- EVENTS - handling container [%s] event %s", event.Status, event.From)
		ev.onStop(event)
	default:
	}

}
