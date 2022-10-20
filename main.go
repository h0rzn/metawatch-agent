package main

import (
	"github.com/h0rzn/monitoring_agent/dock"
)

func main() {
	ctrl, err := dock.NewController()
	defer ctrl.CloseClient()
	if err != nil {
		panic(err)
	}
	containers, err := ctrl.RawContainers()
	if err != nil {
		panic(err)
	}
	for _, engineCont := range containers {
		cont, err := ctrl.Collect(engineCont)
		if err != nil {
			panic(err)
		}
		ctrl.AddContainer(cont)
	}

	whoami := ctrl.Containers[0]
	ctrl.CPU(whoami.ID)

}
