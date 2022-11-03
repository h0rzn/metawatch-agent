package dock

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type Controller struct {
	c          *client.Client
	Containers Storage
}

type Storage struct {
	mutex sync.Mutex
	All   []*Container `json:"containers"`
}

func (sto *Storage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Containers []*Container `json:"containers"`
	}{
		Containers: sto.All,
	})
}

func NewController() (ctrl *Controller, err error) {
	ctrl = new(Controller)
	ctrl.c, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	ctrl.Containers.mutex = sync.Mutex{}

	return ctrl, nil
}

func (ctrl *Controller) CloseClient() {
	ctrl.c.Close()
}

func (ctrl *Controller) ContainersCollect() chan error {
	outErr := make(chan error)

	raw, err := ctrl.rawContainers()
	if err != nil {
		outErr <- err
		close(outErr)
		return outErr
	}

	go func() {
		for _, containerRaw := range raw {
			err := ctrl.registerContainer(containerRaw)
			if err != nil {
				outErr <- err
			}
		}
		close(outErr)

	}()
	return outErr
}

func (ctrl *Controller) rawContainers() ([]types.Container, error) {
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

func (ctrl *Controller) registerContainer(raw types.Container) error {
	cont, err := NewContainer(raw, ctrl.c)
	if err != nil {
		return err
	}

	// check if container exists
	for _, c := range ctrl.Containers.All {
		if c.ID == cont.ID {
			// handle case
			fmt.Printf("ignoring attempt to add container %s [%s], already exists\n", cont.Names, cont.ID)
			return nil
		}
	}

	ctrl.Containers.mutex.Lock()
	ctrl.Containers.All = append(ctrl.Containers.All, cont)
	ctrl.Containers.mutex.Unlock()
	return nil
}

func (ctrl *Controller) Container(id string) *Container {
	for _, container := range ctrl.Containers.All {
		if container.ID == id {
			return container
		}
	}
	return &Container{}
}
