package logs

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Logs struct {
	mutex    *sync.Mutex
	Streamer *stream.Str
	client   *client.Client
	CID      string
}

func NewLogs(c *client.Client, cid string) *Logs {
	return &Logs{
		mutex:    &sync.Mutex{},
		Streamer: nil,
		client:   c,
		CID:      cid,
	}
}

func (l *Logs) Reader() (io.Reader, error) {
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

	r, err := l.client.ContainerLogs(ctx, l.CID, opts)
	if err != nil {
		return r, err
	}
	return r, nil
}

func (l *Logs) Get() *stream.Receiver {
	fmt.Println("[LOGS] GET")
	l.mutex.Lock()
	if l.Streamer == nil {
		err := l.InitStr()
		if err != nil {
			fmt.Println("failed to get receiver for logs")
		}
	}
	l.mutex.Unlock()

	return l.Streamer.Join()
}

func (l *Logs) InitStr() (err error) {
	r, err := l.Reader()
	if err != nil {
		fmt.Println("[LOGS] error initing streamer")
		return
	}

	l.Streamer = stream.NewStr(r)
	go l.Streamer.Run(GenPipe)
	go l.StreamerQuitSig()
	return
}

func (l *Logs) StreamerQuitSig() {
	<-l.Streamer.Quit
	l.Streamer = nil
}

func GenPipe(r io.Reader, done chan struct{}) <-chan stream.Set {
	out := make(chan stream.Set)
	sets := Parse(r, done)

	go func() {
		for set := range sets {
			out <- set
		}
		close(out)
	}()
	return out
}

func Parse(r io.Reader, done chan struct{}) <-chan stream.Set {
	out := make(chan stream.Set)

	go func() {
		for {
			hdr := make([]byte, 8)

			select {
			case <-done:
				close(out)
				return
			default:
			}

			_, err := r.Read(hdr)
			if err != nil {
				if err == io.EOF {
					fmt.Println("eof")
				}
			}

			sizes := binary.BigEndian.Uint32(hdr[4:])
			content := make([]byte, sizes)
			_, err = r.Read(content)
			if err != nil && err != io.EOF {
				//errChan <- err
				fmt.Println(err)
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
