package main

import (
	"github.com/gin-gonic/gin"
	"github.com/h0rzn/monitoring_agent/api"
)

func main() {
	api, err := api.CreateConsumer()
	if err != nil {
		panic(err)
	}
	api.Run()

	r := gin.Default()

	r.GET("/containers/all", api.Containers)
	r.GET("/containers/:id", api.Container)
	r.Run()
}
