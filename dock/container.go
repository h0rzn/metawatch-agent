package dock

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/logs"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
)

type Container struct {
	ID      string          `json:"id"`
	Names   []string        `json:"names"`
	Image   string          `json:"image"`
	State   ContainerState  `json:"state"`
	Metrics MetricsStreamer `json:"-"`
	Logs    *logs.Logs      `json:"-"`
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
	metricsDone := make(chan bool)
	c.Metrics = *NewMetricsStreamer()

	ctx := context.Background()
	conMetr, err := c.c.ContainerStats(ctx, c.ID, true)
	if err != nil {
		panic(err)
	}
	//defer conMetr.Body.Close()
	c.Metrics.Source(conMetr.Body, metricsDone)
	go c.Metrics.Run()

	// register database consumer

	// logs
	c.Logs = logs.NewLogs(c.c, c.ID)

	// start listener for incomming container change information

	return nil
}

func (c *Container) Single(out chan<- *metrics.Set) {
	cons := NewConsumer(true)
	go func(cons *Consumer) {
		c.Metrics.Reg <- cons
		set := <-cons.In
		out <- set
	}(cons)
	close(out)
}

func (c *Container) MetricsStream(done chan bool) <-chan *metrics.Set {
	out := make(chan *metrics.Set)
	go func() {
		cons := NewConsumer(false)
		c.Metrics.Reg <- cons

		for metric := range cons.In {
			select {
			case <-done:
				c.Metrics.Ureg <- cons
			default:
			}
			out <- metric
		}
		close(out)
	}()
	return out
}

func (c *Container) MarshalJSON() ([]byte, error) {
	cons := NewConsumer(true)
	c.Metrics.Reg <- cons
	set := <-cons.In

	type Alias Container
	return json.Marshal(&struct {
		Metrics *metrics.Set `json:"metrics"`
		*Alias
	}{
		Metrics: set,
		Alias:   (*Alias)(c),
	})
}
