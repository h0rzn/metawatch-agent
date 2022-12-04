package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

type Metrics struct {
	mutex    *sync.Mutex
	Streamer *stream.Str
	client   *client.Client
	CID      string
}

func NewMetrics(c *client.Client, cid string) *Metrics {
	return &Metrics{
		mutex:    &sync.Mutex{},
		Streamer: nil,
		client:   c,
		CID:      cid,
	}
}

func (m *Metrics) Reader() (io.ReadCloser, error) {
	ctx := context.Background()
	r, err := m.client.ContainerStats(ctx, m.CID, true)
	return r.Body, err
}

func (m *Metrics) Get(interv bool) *stream.Receiver {
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

func (m *Metrics) InitStr() (err error) {
	r, err := m.Reader()
	if err != nil {
		return
	}

	m.Streamer = stream.NewStr(r)
	go m.Cylce()
	return
}

func (m *Metrics) Cylce() {
	m.Streamer.Run(GenPipe)
	m.Streamer = nil
}

func GenPipe(r io.ReadCloser, done chan struct{}) <-chan stream.Set {
	out := make(chan stream.Set)
	stats := Parse(r, done)

	go func() {
		for stat := range stats {
			metricSet := NewSetWithJSON(stat)
			streamSet := stream.NewSet("metric_set", metricSet)
			out <- *streamSet
		}
		fmt.Println("no longer receiving set")
	}()
	return out
}

func Parse(r io.ReadCloser, done chan struct{}) <-chan types.StatsJSON {
	out := make(chan types.StatsJSON)
	dec := json.NewDecoder(r)

	go func() {
		for {
			select {
			case <-done:
				close(out)
				r.Close()
				return
			default:
			}
			var stat types.StatsJSON
			err := dec.Decode(&stat)
			if err != nil {
				logrus.Warnln("- METRICS - error while decoding, exit parser...")
				return
			}
			out <- stat
		}

	}()
	return out
}
