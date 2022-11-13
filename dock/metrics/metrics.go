package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Metrics struct {
	mutex    sync.Mutex
	C        *client.Client
	CID      string
	Streamer stream.Streamer
	Done     chan interface{}
}

func NewMetrics(c *client.Client, cid string) *Metrics {
	return &Metrics{
		mutex: sync.Mutex{},
		C:     c,
		CID:   cid,
		Done:  make(chan interface{}),
	}
}

func (m *Metrics) Source() (io.Reader, error) {
	ctx := context.Background()
	r, err := m.C.ContainerStats(ctx, m.CID, true)
	return r.Body, err
}

func (m *Metrics) Subscribe() (stream.Subscriber, error) {
	m.mutex.Lock()
	if m.Streamer == nil {
		r, err := m.Source()
		if err != nil {
			return &stream.TempSub{}, errors.New("error creating log reader")
		}
		m.Streamer = NewStreamer(r)
		go m.Streamer.Run()
	}

	m.mutex.Unlock()
	return m.Streamer.Subscribe(), nil
}

func (m *Metrics) Stream(done chan interface{}) chan *stream.Set {
	out := make(chan *stream.Set)
	sub, err := m.Subscribe()
	if err != nil {
		fmt.Println("failed to subscribe to metrics")
	}

	go func() {
		go sub.Handle(out)
		<-done
		m.Streamer.Unsubscribe(sub)
	}()
	return out
}
