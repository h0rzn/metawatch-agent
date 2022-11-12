package logs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Logs struct {
	mutex    sync.Mutex
	C        *client.Client
	CID      string
	Streamer *Streamer
	Done     chan bool
}

func NewLogs(c *client.Client, cid string) *Logs {
	return &Logs{
		mutex: sync.Mutex{},
		C:     c,
		CID:   cid,
		Done:  make(chan bool),
	}
}

func (l *Logs) subscribe() (*Sub, error) {
	l.mutex.Lock()
	if l.Streamer == nil {
		r, err := l.reader()
		if err != nil {
			return &Sub{}, errors.New("error creating log reader")
		}
		l.Streamer = NewStreamer(r)
		go l.Streamer.Run()
	}

	l.mutex.Unlock()
	return l.Streamer.Sub(), nil
}

func (l *Logs) reader() (io.Reader, error) {
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

func (l *Logs) Stream(done chan bool) chan *Entry {
	fmt.Println("requesting log stream")

	out := make(chan *Entry)
	sub, err := l.subscribe()
	if err != nil {
		fmt.Println("failed to subscribe to logs")
	}

	go func() {
		go sub.Handle(out)
		b := <-done
		fmt.Println("done rcvd", b)
		l.Streamer.USub(sub)
	}()

	return out
}
