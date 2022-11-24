package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/api/hub"
	"github.com/h0rzn/monitoring_agent/dock/controller"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type API struct {
	Router     *gin.Engine
	Addr       string
	Controller *controller.Controller
	Hub        *hub.Hub
}

func NewAPI(addr string) (*API, error) {
	ctrl, err := controller.NewController()
	if err != nil {
		return &API{}, err
	}
	return &API{
		Router:     gin.Default(),
		Addr:       addr,
		Controller: ctrl,
		Hub:        hub.NewHub(ctrl),
	}, nil
}

func corsMW(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET")
}

func (api *API) RegRoutes() {
	api.Router.Use(corsMW)
	api.Router.GET("/containers/:id", api.Container)
	api.Router.GET("/containers/all", api.Containers)
	api.Router.GET("/containers/:id/stream", api.ContainerMetrics)
	api.Router.GET("/containers/:id/logs", api.ContainerLogs)
	api.Router.GET("/test", api.Test)
}

func (api *API) Run() {
	api.Controller.Init()
	go api.Hub.Run()
	api.Router.Run(api.Addr)

}
