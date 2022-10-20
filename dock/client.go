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

func (ctrl *Controller) Collect(engineCont types.Container) (*Container, error) {
	var cont Container
	cont.Names = engineCont.Names
	cont.Image = engineCont.Image
	cont.ID = engineCont.ID

	// get container state over types.ContainerJson.ContainerJSONBase
	ctx := context.Background()
	json, err := ctrl.c.ContainerInspect(ctx, engineCont.ID)
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
	cont.State = state

	return &cont, nil
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
	fmt.Println(len(ctrl.Containers), cont.Names, cont.State.Status, cont.State.Started)
}

func (ctrl *Controller) CPU(id string) {
	ctx := context.Background()
	contStats, err := ctrl.c.ContainerStats(ctx, id, true)
	if err != nil {
		panic(err)
	}
	defer contStats.Body.Close()

	whoami := ctrl.Containers[0]
	whoami.ReadStats(contStats.Body)

}
