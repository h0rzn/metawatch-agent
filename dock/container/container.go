package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/logs"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/sirupsen/logrus"
)

const feederInterv = 5 * time.Second

type Container struct {
	ID      string
	Names   []string
	Image   string
	State   State
	Ports   []string
	Streams Streams
	c       *client.Client
}

type State struct {
	Status  string `json:"status"`
	Started string `json:"since"`
}

type Streams struct {
	Logs       *logs.Logs
	Metrics    *metrics.Metrics
	FeederDone chan struct{}
	FeedOut    chan interface{}
}

// Feed collects data from streamer and feeds them to tb
func (s *Streams) Feed(done chan struct{}, cid string) {
	metricsRcv, err := s.Metrics.Get(true)
	if err != nil {
		return
	}

	feederTicker := time.NewTicker(feederInterv)
	for {
		select {
		case <-done:
			return
		case <-metricsRcv.Closing:
			fmt.Println("feeder: rcv closing sig")
			return
		case <-feederTicker.C:
			set, ok := <-metricsRcv.In
			if !ok {
				fmt.Println("feeder: rcv in is closed")
				return
			}
			if metricsSet, ok := set.Data.(metrics.Set); ok {
				metricsWrap := db.NewMetricsMod(cid, metricsSet.When, metricsSet)
				s.FeedOut <- metricsWrap
				logrus.Tracef("- Container - -> sending for cid: %s\n", metricsWrap.CID)

			} else {
				logrus.Info("- LINK - failed to parse incomming interface to metrics.Set")
			}
		}
	}
}

func (s *Streams) Stop() error {
	logrus.Debugln("- CONTAINER - stop: stopping streams and feeder")
	s.FeederDone <- struct{}{}

	err := s.Metrics.Stop()
	if err != nil {
		return err
	}

	err = s.Logs.Stop()
	if err != nil {
		return err
	}

	return nil
}

type ContainerJSON struct {
	ID      string       `json:"id"`
	Names   []string     `json:"names"`
	Image   string       `json:"image"`
	State   State        `json:"state"`
	Ports   []string     `json:"ports"`
	Metrics *metrics.Set `json:"metrics"`
}

func NewContainer(raw types.Container, c *client.Client, feedOut chan interface{}) *Container {
	ports := make([]string, 0)
	for _, p := range raw.Ports {
		pFmt := fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
		ports = append(ports, pFmt)
	}

	return &Container{
		ID:    raw.ID,
		Names: raw.Names,
		Image: raw.Image,
		Ports: ports,
		Streams: Streams{
			Metrics:    metrics.NewMetrics(c, raw.ID),
			Logs:       logs.NewLogs(c, raw.ID),
			FeederDone: make(chan struct{}),
			FeedOut:    feedOut,
		},
		c: c,
	}
}

func (cont *Container) prepare() <-chan error {
	out := make(chan error, 1)

	// inspect container for further info
	ctx := context.Background()
	json, err := cont.c.ContainerInspect(ctx, cont.ID)
	if err != nil {
		fmt.Println("inspect err", err)
		out <- err
		return out
	}

	jsonBase := json.ContainerJSONBase
	status := jsonBase.State.Status
	statusStarted := jsonBase.State.StartedAt

	// set current state
	cont.State = State{
		Status:  status,
		Started: statusStarted,
	}

	// start feeder
	go cont.Streams.Feed(cont.Streams.FeederDone, cont.ID)

	out <- err
	return out
}

func (cont *Container) Start() error {
	logrus.Infoln("- CONTAINER - preparing...")
	select {
	case err := <-cont.prepare():
		return err
	case <-time.After(8 * time.Second):
		return errors.New("container start timed out")
	}
}

func (cont *Container) Stop() error {
	logrus.Infoln("- CONTAINER - stop")
	return cont.Streams.Stop()
}

// JSON returns a json valid struct for a container
func (cont *Container) JSONSkel() *ContainerJSON {
	var currentMetrics metrics.Set

	recv, err := cont.Streams.Metrics.Get(false)
	if err != nil {
		return &ContainerJSON{}
	}
	for cur := range recv.In {
		currentMetrics = cur.Data.(metrics.Set)
		break
	}
	recv.Close()

	return &ContainerJSON{
		ID:      cont.ID,
		Names:   cont.Names,
		Image:   cont.Image,
		State:   cont.State,
		Ports:   cont.Ports,
		Metrics: &currentMetrics,
	}
}
