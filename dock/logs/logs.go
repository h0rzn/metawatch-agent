package logs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

// Logs implements stream.Director interface
type Logs struct {
	mutex    sync.Mutex
	C        *client.Client
	CID      string
	Streamer stream.Streamer
	Done     chan struct{}
}

func NewLogs(c *client.Client, cid string) *Logs {
	return &Logs{
		mutex: sync.Mutex{},
		C:     c,
		CID:   cid,
		Done:  make(chan struct{}),
	}
}

// Source fetches new logs reader
func (l *Logs) Source() (io.Reader, error) {
	ctx := context.Background()
	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		//Since: unix timestamp, go duration string (1m30s)
		Timestamps: true,
		//Details:    true,
		Follow: true,
		Tail:   "0",
	}

	r, err := l.C.ContainerLogs(ctx, l.CID, opts)
	if err != nil {
		return r, err
	}
	return r, nil
}

// Subscribe returns new Subscriber
// if streamer is nil, a new streamer will be created in addition
func (l *Logs) Subscribe() (stream.Subscriber, error) {
	l.mutex.Lock()
	if l.Streamer == nil {
		r, err := l.Source()
		if err != nil {
			return &stream.TempSub{}, errors.New("error creating log reader")
		}
		l.Streamer = NewStreamer(r)
		go l.Streamer.Run()
	}

	l.mutex.Unlock()
	return l.Streamer.Subscribe(), nil
}

// Stream returns the log entries respectively as a set through a returned channel
func (l *Logs) Stream(done chan struct{}) chan *stream.Set {
	fmt.Println("requesting log stream")

	out := make(chan *stream.Set)
	sub, err := l.Subscribe()
	if err != nil {
		fmt.Println("failed to subscribe to logs")
	}

	go func() {
		go sub.Handle(out)
		<-done
		l.Streamer.Unsubscribe(sub)
	}()

	return out
}
