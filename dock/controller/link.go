package controller

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

const LinkSendInterv time.Duration = 5 * time.Second

type Link struct {
	ContainerID string
	Done        chan struct{}
	Metrics     *stream.Receiver
	// add events

	// out channel for results
	Out chan db.WriteSet
}

func NewLink(cid string) *Link {
	return &Link{
		ContainerID: cid,
		Done:        make(chan struct{}),
		Out:         make(chan db.WriteSet),
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
			// handle quit
		case set, ok := <-l.Metrics.In:
			if !ok {
				fmt.Println("[LINK] cannot read: channel closed")
				// handle closed channel error
			}
			select {
			case <-ticker.C:
				fmt.Println("[LINK] 'tick', sending", set)
				setJson, err := json.Marshal(set)
				if err != nil {
					fmt.Println("[LINK] json marshal error", err)
				}

				writeSet := db.WriteSet{
					ContainerID: l.ContainerID,
					Metrics:     setJson,
				}

				l.Out <- writeSet
			default:
				_ = set
			}
		}
	}
}
