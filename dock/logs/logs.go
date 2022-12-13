package logs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"github.com/sirupsen/logrus"
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

func (l *Logs) Reader() (io.ReadCloser, error) {
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

func (l *Logs) Get(interv bool) (*stream.Receiver, error) {
	logrus.Infoln("- LOGS - requested receiver")
	l.mutex.Lock()
	if l.Streamer == nil {
		err := l.InitStr()
		if err != nil {
			logrus.Errorln("- LOGS - failed to get receiver")
		}
	}
	l.mutex.Unlock()

	return l.Streamer.Join(interv)
}

func (l *Logs) InitStr() (err error) {
	r, err := l.Reader()
	if err != nil {
		logrus.Errorln("- LOGS - error initing streamer")
		return
	}

	pipe := NewPipeline(r)
	l.Streamer = stream.NewStr(pipe)
	go l.Streamer.Run()
	return
}

func (l *Logs) Stop() error {
	logrus.Debugln("- LOGS - stopping...")
	if l.Streamer == nil {
		return nil
	}

	clsErr := l.Streamer.Cls()

	// handle closing error
	select {
	case err := <-clsErr:
		if err != nil {
			logrus.Errorf("- LOGS - streamer close err: %s\n", err)
		}
		fmt.Println("metrics closed")
		l.Streamer = nil
		return err
	case <-time.After(10 * time.Second):
		l.Streamer = nil
		return errors.New("logs stop timeout")
	}
}
