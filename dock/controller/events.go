package controller

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Events struct {
	c    *client.Client
	Strg *Storage
}

func NewEvents(c *client.Client, strg *Storage) *Events {
	return &Events{
		c:    c,
		Strg: strg,
	}
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
			_ = err
		}
	}

}

func (ev *Events) onStop(e events.Message) {
	// check if container is indexed
	if container, exists := ev.Strg.Container(e.ID); exists {
		ev.Strg.Remove(container)
	} else {
		logrus.Debugln("- EVENTS - stop: container not indexed")
	}
}

// https://docs.docker.com/engine/reference/commandline/events/
func (ev *Events) catch(event events.Message) {
	// only handle container events for now
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
	// case "start":
	// 	logrus.Infof("- EVENTS - handling container [%s] event %s", event.Status, event.From)
	// 	ev.onStart(event)
	case "stop":
		logrus.Infof("- EVENTS - handling container [%s] event %s", event.Status, event.From)
		ev.onStop(event)
	default:
	}

}
