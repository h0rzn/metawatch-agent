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
	fmt.Println("preparing container")
	select {
	case err := <-cont.prepare():
		return err
	case <-time.After(8 * time.Second):
		fmt.Printf("container %s start timed out\n", cont.Names[0])
		return errors.New("container start timed out")
	}
}

// JSON returns a json valid struct for a container
func (c *Container) JSONSkel() *ContainerJSON {
	var currentMetrics metrics.Set

	recv := c.Streams.Metrics.Get()
	for cur := range recv.In {
		currentMetrics = cur.Data.(metrics.Set)
		break
	}
	recv.Quit()

	return &ContainerJSON{
		ID:      c.ID,
		Names:   c.Names,
		Image:   c.Image,
		State:   c.State,
		Metrics: &currentMetrics,
	}
}
