package controller

import (
	"fmt"

	"time"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

const LinkSendInterv time.Duration = 5 * time.Second

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

func (l *Link) Init(container *container.Container) {
	l.Metrics = container.Streams.Metrics.Get()
}

func (l *Link) Run() {
	ticker := time.NewTicker(LinkSendInterv)

	for {
		select {
		case <-l.Done:
			l.Metrics.Quit()
			return
		case set, ok := <-l.Metrics.In:
			if !ok {
				fmt.Println("[LINK] cannot read: channel closed")
				return
			}
			select {
			case <-ticker.C:
				if metricsSet, ok := set.Data.(metrics.Set); ok {
					metricsWrap := db.NewMetricsMod(l.ContainerID, metricsSet.When, metricsSet)
					l.Out <- metricsWrap
					fmt.Println("[LINK] 'tick', sent:", metricsWrap)

				} else {
					fmt.Println("[LINK] failed to parse incomming interface to metrics.Set")
				}
			default:
				_ = set
			}
		}
	}
}
