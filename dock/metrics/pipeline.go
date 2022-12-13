package metrics

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Pipeline struct {
	R    io.ReadCloser
	done chan struct{}
}

func NewPipeline(r io.ReadCloser) *Pipeline {
	return &Pipeline{
		R: r,
		done: make(chan struct{}, 2),
	}
}

func (p *Pipeline) parse() chan types.StatsJSON {
	out := make(chan types.StatsJSON)
	dec := json.NewDecoder(p.R)

	go func() {
		defer close(out)
		for {
			select {
			case <-p.done:
				fmt.Println("metrics pipeline: parse done")
				return
			default:
			}
			var stat types.StatsJSON
			err := dec.Decode(&stat)
			if err != nil {
				return
			}
			out <- stat
		}

	}()
	return out

}

func (p *Pipeline) toSet(parsed chan types.StatsJSON) chan stream.Set {
	out := make(chan stream.Set)
	go func() {
		defer close(out)
		for in := range parsed {
			set := stream.NewSet("metrics_set", NewSetWithJSON(in))
			select {
			case out <- *set:
			case <-p.done:
				fmt.Println("metrics pipeline: toSet done")
				return
			}
		}
	}()
	return out
}

func (p *Pipeline) Out() chan stream.Set {
	return p.toSet(p.parse())
}

func (p *Pipeline) Stop() {
	p.done <- struct{}{}
	p.done <- struct{}{}
	p.R.Close()
	close(p.done)
}
