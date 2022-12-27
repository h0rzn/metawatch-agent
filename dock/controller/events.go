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
	Queue  chan events.Message
	Inform func(events.Message)
}

func NewEvents(c *client.Client, strg *Storage) *Events {
	return &Events{
		c:     c,
		Strg:  strg,
		Queue: make(chan events.Message),
	}
}

func (ev *Events) SetInformer(fn func(events.Message)) {
	ev.Inform = fn
}

func (ev *Events) onStop(e events.Message) {
	err := ev.Strg.ContainerStore.Stop(e.ID)
	if err != nil {
		logrus.Errorf("- EVENTS - failed to handle [stop] event: %s\n", err)
	} else {
		logrus.Infoln("- EVENTS - succesfully removed container based on [stop]")
	}
	ev.Inform(e)
}

func (ev *Events) onDestroy(e events.Message) {
	err := ev.Strg.ContainerStore.Remove(e.ID)
	if err != nil {
		logrus.Errorf("- EVENTS - failed to handle [destroy] event: %s\n", err)
	} else {
		logrus.Infoln("- EVENTS - succesfully removed container based on [destroy]")
	}
	ev.Inform(e)
}

func (ev *Events) onStart(e events.Message) {
	err := ev.Strg.ContainerStore.Add(e.ID)
	if err != nil {
		logrus.Errorf("- EVENTS - failed to start container, [start] event: %s\n", err)
	}
	logrus.Infoln("- EVENTS - succesfully started container based on [start]")
	ev.Inform(e)
}

// https://docs.docker.com/engine/reference/commandline/events/
func (ev *Events) dispatchContainerEvent(event events.Message) {
	logrus.Infoln("//////// EVENT:", event.Status)

	switch event.Status {
	case "start":
		logrus.Infof("- EVENTS - handling container [%s] event %s", event.Status, event.From)
		go ev.onStart(event)

	case "stop":
		logrus.Infof("- EVENTS - handling container [%s] event %s", event.Status, event.From)
		go ev.onStop(event)

	case "destroy":
		// remove container
		go ev.onDestroy(event)
	default:
	}
}

func (ev *Events) channelQueue(in chan events.Message, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case event := <-in:
			logrus.Debugf("- EVENTS - %s- event queued\n", event.Type)
			switch event.Type {
			case events.ContainerEventType:
				ev.dispatchContainerEvent(event)
				
			}
		}
	}
}

func (ev *Events) Run() {
	logrus.Infoln("- EVENTS - running...")
	ctx := context.Background()
	evs, errs := ev.c.Events(ctx, types.EventsOptions{})

	push := make(chan events.Message)
	done := make(chan struct{})
	go ev.channelQueue(push, done)

	for {
		select {
		case event := <-evs:
			switch event.Type {
			case events.ContainerEventType:
				push <- event

			default:
			}

		case err := <-errs:
			logrus.Errorf("- EVENTS - erroring receiving events: %s\n", err.Error())
			done <- struct{}{}
			return
		}
	}

}
