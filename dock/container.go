package dock

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/logs"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Container struct {
	ID      string          `json:"id"`
	Names   []string        `json:"names"`
	Image   string          `json:"image"`
	State   ContainerState  `json:"state"`
	Metrics stream.Director `json:"-"`
	Logs    stream.Director `json:"-"`
	c       *client.Client  `json:"-"`
}

type ContainerState struct {
	Status  string `json:"status"` // "running", "stopped", ...
	Started string `json:"since"`
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

	// fetch port bindings

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

func (c *Container) Init() error {
	// metrics
	c.Metrics = metrics.NewMetrics(c.c, c.ID)
	c.Metrics.Source()

	// register database consumer

	// logs
	c.Logs = logs.NewLogs(c.c, c.ID)

	// start listener for incomming container change information

	return nil
}

func (c *Container) MarshalJSON() ([]byte, error) {
	done := make(chan interface{})
	metricC := c.Metrics.Stream(done)
	set := <-metricC
	done <- true

	n, ok := set.Data.(*metrics.Set)
	if !ok {
		return nil, errors.New("failed to assert interface{} to *metric.Set when marshalling")
	}

	type Alias Container
	return json.Marshal(&struct {
		Metrics *metrics.Set `json:"metrics"`
		*Alias
	}{
		Metrics: n,
		Alias:   (*Alias)(c),
	})
}
