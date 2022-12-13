package controller

import (
	"fmt"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

type Link struct {
	ContainerID string
	Done        chan struct{}
	Metrics     *stream.Receiver
	// add events

	// out channel for results
	Out chan interface{}
}

func NewLink(cid string) *Link {
	return &Link{
		ContainerID: cid,
		Done:        make(chan struct{}),
		Out:         make(chan interface{}),
	}
}

func (l *Link) Init(container *container.Container) error {
	recv, err := container.Streams.Metrics.Get(true)
	if err != nil {
		return err
	}
	l.Metrics = recv
	return nil
}

func (l *Link) Run() {
	for {
		select {
		case <-l.Done:
			l.Metrics.Close(false)
			close(l.Out)
			return
		case <-l.Metrics.Closing:
			fmt.Println("link receiver closing sig")
			close(l.Out)
			return
		case set, ok := <-l.Metrics.In:
			if !ok {
				logrus.Errorln("- LINK - cannot read: channel closed")
				close(l.Out)
				return
			}
			if metricsSet, ok := set.Data.(metrics.Set); ok {
				metricsWrap := db.NewMetricsMod(l.ContainerID, metricsSet.When, metricsSet)
				l.Out <- metricsWrap
				logrus.Debugf("- LINK - tick -> sending for cid: %s\n", metricsWrap.CID)

			} else {
				logrus.Info("- LINK - failed to parse incomming interface to metrics.Set")
			}

		}

	}
}
