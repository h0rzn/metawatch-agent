package api

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/api/hub"
	"github.com/h0rzn/monitoring_agent/dock/controller"
	"github.com/sirupsen/logrus"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type API struct {
	Router     *gin.Engine
	Addr       string
	Controller *controller.Controller
	Hub        *hub.Hub
}

func NewAPI() (*API, error) {
	ctrl, err := controller.NewController()
	if err != nil {
		return &API{}, err
	}

	addr := os.Getenv("ADDR")
	if addr == "" {

		addr = "localhost:8080"
		logrus.Infoln("ADDR not specified: running on default localhost:8080")
	}

	return &API{
		Router:     gin.Default(),
		Addr:       addr,
		Controller: ctrl,
		Hub:        hub.NewHub(ctrl),
	}, nil
}

func (api *API) RegRoutes() error {
	jwt, err := JWT(api.Controller.DB.PasswordCorrect)
	if err != nil {
		return err
	}

	api.Router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowHeaders:     []string{"Authorization", "content-type"},
		AllowMethods:     []string{"PUT", "PATCH", "DELETE", "GET", "POST"},
		AllowCredentials: true,
	}))

	api.Router.POST("/login", jwt.LoginHandler)
	authed := api.Router.Group("/api")
	authed.Use(jwt.MiddlewareFunc())
	authed.GET("refresh_token", jwt.RefreshHandler)

	authed.GET("/containers/:id", api.Container)
	authed.GET("/containers/all", api.Containers)
	authed.GET("/containers/:id/metrics", api.Metrics)
	authed.GET("/images", api.Images)
	authed.GET("/images/:id", api.Image)
	authed.GET("/about", api.About)
	authed.GET("/volumes", api.Volumes)

	authed.POST("/users", api.RegisterUser)
	authed.DELETE("/users/:id", api.RemoveUser)
	authed.GET("/users", api.GetUsers)
	authed.PATCH("/users/:id", api.PatchUser)

	api.Router.GET("/stream", api.Stream)

	return nil
}

func (api *API) Run() {
	err := api.Controller.Init()
	if err != nil {
		logrus.Errorln("- API - failed to create controller, leaving...")
		return
	}
	go api.Hub.Run()
	// api.Controller.Storage.Events.SetInformer(api.Hub.BroadcastEvent)
	logrus.Infoln("- API - starting gin router")
	api.Router.Run(api.Addr)
}
