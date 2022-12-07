package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/logs"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/sirupsen/logrus"
)

type Container struct {
	ID      string
	Names   []string
	Image   string
	State   State
	Streams Streams
	c       *client.Client
}

type State struct {
	Status  string `json:"status"`
	Started string `json:"since"`
}

type Streams struct {
	Logs    *logs.Logs
	Metrics *metrics.Metrics
}

type ContainerJSON struct {
	ID      string       `json:"id"`
	Names   []string     `json:"names"`
	Image   string       `json:"image"`
	State   State        `json:"state"`
	Metrics *metrics.Set `json:"metrics"`
}

func NewContainer(raw types.Container, c *client.Client) *Container {
	return &Container{
		ID:    raw.ID,
		Names: raw.Names,
		Image: raw.Image,
		Streams: Streams{
			Metrics: metrics.NewMetrics(c, raw.ID),
			Logs:    logs.NewLogs(c, raw.ID),
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

func (cont *Container) Stop() {
	err := cont.Streams.Metrics.Stop()
	if err != nil {
		logrus.Errorf("- CONTAINER - stop err: %s", err)
	}
	err = cont.Streams.Logs.Stop()
	if err != nil {
		logrus.Errorf("- CONTAINER - stop err: %s", err)
	}
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
	recv.Close(false)

	return &ContainerJSON{
		ID:      cont.ID,
		Names:   cont.Names,
		Image:   cont.Image,
		State:   cont.State,
		Metrics: &currentMetrics,
	}
}
