package dock

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
)

type Container struct {
	ID      string
	Names   []string
	Image   string
	State   ContainerState
	Metrics MetricsStreamer
	Update  chan interface{}
	c       *client.Client
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

	// start listener for incomming container change information

	return nil
}

func (c *Container) MetricsSingle() <-chan *metrics.Set {
	out := make(chan *metrics.Set, 1) // single set will be sent
	cons := NewConsumer(true)
	c.Metrics.Reg <- cons

	set := <-cons.In
	out <- set
	close(out)
	return out
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

func (c *Container) Run() {
	c.Metrics.Run()
}
