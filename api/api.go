package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/dock"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type API struct {
	Router     *gin.Engine
	Addr       string
	Controller *dock.Controller
}

func NewAPI(addr string) (*API, error) {
	ctrl, err := dock.NewController()
	if err != nil {
		return &API{}, err
	}
	return &API{
		Router:     gin.Default(),
		Addr:       addr,
		Controller: ctrl,
	}, nil
}

func (api *API) RegRoutes() {
	api.Router.GET("/containers/:id", api.Container)
	api.Router.GET("containers/all", api.Containers)
	api.Router.GET("/containers/:id/stream", api.ContainerMetrics)
}

func (api *API) Run() {
	errorChan := api.Controller.ContainersCollect()
	for err := range errorChan {
		fmt.Println(err.Error())
	}
	for _, container := range api.Controller.Containers.All {
		go func(c *dock.Container) {
			err := c.Init()
			if err != nil {
				panic(err)
			}
		}(container)
	}

	api.Router.Use(ContentType())
	api.Router.Run(api.Addr)
}

func ContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Next()
	}
}
