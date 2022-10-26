package dock

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type Controller struct {
	c          *client.Client
	Containers []*Container
}

func NewController() (ctrl *Controller, err error) {
	ctrl = new(Controller)
	ctrl.c, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return ctrl, nil
}

func (ctrl *Controller) CloseClient() {
	ctrl.c.Close()
}

func (ctrl *Controller) RawContainers() ([]types.Container, error) {
	ctx := context.Background()
	containers, err := ctrl.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: filters.Args{},
		})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (ctrl *Controller) Collect(engineCont types.Container) error {
	container, err := NewContainer(engineCont, ctrl.c)
	if err != nil {
		return err
	}

	ctrl.AddContainer(container)
	return nil
}

func (ctrl *Controller) AddContainer(cont *Container) {
	// check if container exists
	for _, c := range ctrl.Containers {
		if c.ID == cont.ID {
			// handle case
			fmt.Printf("error adding container %s, already exists\n", cont.Names)
			return
		}
	}

	ctrl.Containers = append(ctrl.Containers, cont)
}
