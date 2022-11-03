package main

import (
	"github.com/h0rzn/monitoring_agent/api"
)

func main() {
	api, err := api.NewAPI(":8080")
	if err != nil {
		panic(err)
	}
	api.RegRoutes()
	api.Run()
}
