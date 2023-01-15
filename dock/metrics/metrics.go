package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

type Metrics struct {
	mutex     *sync.Mutex
	Streamer  *stream.Str
	client    *client.Client
	CID       string
	LatestSet Set
	LatestRcv *stream.Receiver
}

func NewMetrics(c *client.Client, cid string) *Metrics {
	return &Metrics{
		mutex:    &sync.Mutex{},
		Streamer: nil,
		client:   c,
		CID:      cid,
	}
}

func (m *Metrics) Init() error {
	// create persistent receiver: "caching layer"
	rcv, err := m.Get(true)
	if err != nil {
		return err
	}
	m.LatestRcv = rcv
	go m.HandleLatest()

	return nil
}

func (m *Metrics) Reader() (io.ReadCloser, error) {
	ctx := context.Background()
	r, err := m.client.ContainerStats(ctx, m.CID, true)
	return r.Body, err
}

func (m *Metrics) InitStr() (err error) {
	r, err := m.Reader()
	if err != nil {
		return
	}
	pipe := NewPipeline(r)
	m.Streamer = stream.NewStr(pipe)
	go m.Streamer.Run()
	return
}

func (m *Metrics) Get(interv bool) (*stream.Receiver, error) {
	logrus.Infoln("- METRICS - requested receiver")
	m.mutex.Lock()
	if m.Streamer == nil {
		err := m.InitStr()
		if err != nil {
			logrus.Errorln("- METRICS - failed to get receiver")
		}
	}
	m.mutex.Unlock()

	return m.Streamer.Join(interv)
}

func (m *Metrics) HandleLatest() {
	fmt.Println("metrics: handling latest now")
	for set := range m.LatestRcv.In {
		// fmt.Println("metrics handle latest: handle set", m.CID)
		metrics, ok := (set.Data).(Set)
		if ok {
			m.mutex.Lock()
			m.LatestSet = metrics
			m.mutex.Unlock()
		}
	}
}

func (m *Metrics) Latest() Set {
	return m.LatestSet
}

func (m *Metrics) Stop() error {
	logrus.Debugln("- METRICS - stopping...")
	if m.Streamer == nil {
		return nil
	}
	clsErr := m.Streamer.Cls()

	// handle closing error
	select {
	case err := <-clsErr:
		if err != nil {
			logrus.Errorf("- METRICS - streamer close err: %s\n", err)
		}
		m.Streamer = nil
		return err
	case <-time.After(10 * time.Second):
		m.Streamer = nil
		return errors.New("metrics stop timeout")
	}
}
