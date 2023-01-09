package events

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

type Events struct {
	mutex    *sync.Mutex
	c        *client.Client
	Streamer *stream.Str
}

func NewEvents(c *client.Client) *Events {
	return &Events{
		mutex: &sync.Mutex{},
		c:     c,
	}
}

func (e *Events) Init() error {
	err := e.InitStr()
	if err != nil {
		logrus.Errorf("- EVENTS - failed to init: %s\n", err)
		return err
	}
	return nil
}

func (e *Events) Reader() (<-chan events.Message, <-chan error) {
	ctx := context.Background()
	evs, errs := e.c.Events(ctx, types.EventsOptions{})
	return evs, errs
}

func (e *Events) InitStr() (err error) {
	r, errs := e.Reader()
	pipe := NewPipeline(r, errs)
	e.Streamer = stream.NewStr(pipe)
	go e.Streamer.Run()
	return
}

func (e *Events) Get() (*stream.Receiver, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.Streamer == nil {
		err := e.InitStr()
		if err != nil {
			return &stream.Receiver{}, err
		}
	}
	return e.Streamer.Join(false)
}
