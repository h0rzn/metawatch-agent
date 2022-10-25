package dock

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/stats"
)

type Container struct {
	ID    string
	Names []string
	Image string
	State ContainerState
	c     *client.Client
}

type ContainerState struct {
	Status  string // "running", "stopped", ...
	Started string
}

func NewContainer(engineC types.Container, c *client.Client) (*Container, error) {
	var container Container
	container.Names = engineC.Names
	container.Image = engineC.Image
	container.ID = engineC.ID

	// fetch container state
	ctx := context.Background()
	json, err := c.ContainerInspect(ctx, engineC.ID)
	if err != nil {
		return &Container{}, err
	}
	jsonBase := json.ContainerJSONBase
	status := jsonBase.State.Status
	statusStarted := jsonBase.State.StartedAt

	state := ContainerState{
		Status:  status,
		Started: statusStarted,
	}
	container.State = state

	// reference received docker client reference
	container.c = c

	return &container, nil
}

func (c *Container) StatsSnap() stats.Set {
	ctx := context.Background()
	contStats, err := c.c.ContainerStats(ctx, c.ID, false)
	if err != nil {
		panic(err)
	}
	defer contStats.Body.Close()

	return stats.NewSet(contStats.Body)
}

func (c *Container) StatsStream() <-chan stats.Set {
	ctx := context.Background()
	contStats, err := c.c.ContainerStats(ctx, c.ID, true)
	if err != nil {
		panic(err)
	}
	defer contStats.Body.Close()

	dec := json.NewDecoder(contStats.Body)
	prod := stats.ProduceStats(dec)
	out := stats.GetFromStream(prod)
	return out
}
