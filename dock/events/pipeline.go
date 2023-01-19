package events

import (
	"github.com/docker/docker/api/types/events"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
)

type Pipeline struct {
	Events <-chan events.Message
	Errors <-chan error
	done   chan struct{}
}

func NewPipeline(evs <-chan events.Message, errs <-chan error) *Pipeline {
	return &Pipeline{
		Events: evs,
		Errors: errs,
		done:   make(chan struct{}, 2),
	}
}

func (p *Pipeline) fetch() chan events.Message {
	out := make(chan events.Message)
	go func() {
		for {
			select {
			case <-p.done:
				return
			case event := <-p.Events:
				out <- event
			case err := <-p.Errors:
				logrus.Errorf("- EVENTS - %s, exit", err)
				return
			}
		}
	}()
	return out
}

func (p *Pipeline) parse(messages chan events.Message) chan stream.Set {
	out := make(chan stream.Set)
	go func() {
		for ev := range messages {
			set := stream.Set{
				Type: "event",
				Data: ev,
			}
			out <- set
		}
	}()

	return out
}

func (p *Pipeline) Out() chan stream.Set {
	return p.parse(p.fetch())
}

func (p *Pipeline) Stop() {
	p.done <- struct{}{}
	p.done <- struct{}{}
	close(p.done)
}
