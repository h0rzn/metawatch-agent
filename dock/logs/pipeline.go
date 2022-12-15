package logs

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Pipeline struct {
	R    io.ReadCloser
	done chan struct{}
}

func NewPipeline(r io.ReadCloser) *Pipeline {
	return &Pipeline{
		R:    r,
		done: make(chan struct{}, 1),
	}
}

func (p *Pipeline) parse() chan stream.Set {
	out := make(chan stream.Set)
	go func() {
		defer close(out)
		for {
			hdr := make([]byte, 8)

			select {
			case <-p.done:
				fmt.Println("logs parser: done")
				return
			default:
			}

			_, err := p.R.Read(hdr)
			if err != nil {
				return
			}

			sizes := binary.BigEndian.Uint32(hdr[4:])
			content := make([]byte, sizes)
			_, err = p.R.Read(content)
			if err != nil {
				return
			}
			time, data, found := strings.Cut(string(content), " ")
			if found {
				entry := NewEntry(time, data, hdr[0])
				set := stream.NewSet("log_entry", entry)
				out <- *set
			}
		}
	}()
	return out
}

func (p *Pipeline) Out() chan stream.Set {
	return p.parse()
}

func (p *Pipeline) Stop() {
	p.done <- struct{}{}
	p.R.Close()
	close(p.done)
}
