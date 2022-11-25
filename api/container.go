package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type KeepAliveMsg struct {
	KeepAlive bool `json:"keep_alive"`
}

func (api *API) Container(ctx *gin.Context) {
	id := ctx.Param("id")
	cont, exists := api.Controller.ContainerGet(id)
	if !exists {
		HttpErr(ctx, http.StatusNotFound, errors.New("container not found"))
	}
	json := cont.JSONSkel()
	ctx.JSON(http.StatusOK, json)
}

func (api *API) Containers(ctx *gin.Context) {
	json := api.Controller.Storage.JSONSkel()
	ctx.JSON(http.StatusOK, json)
}

func (api *API) ContainerMetrics(ctx *gin.Context) {
	id := ctx.Param("id")
	api.metricsWS(ctx.Writer, ctx.Request, id)
}

func (api *API) ContainerLogs(ctx *gin.Context) {
	id := ctx.Param("id")
	api.logsWS(ctx.Writer, ctx.Request, id)
}

func (api *API) metricsWS(w http.ResponseWriter, r *http.Request, id string) {
	con, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		w.Write(errBytes)
		return
	}

	container, exists := api.Controller.ContainerGet(id)
	if !exists {
		errBytes, _ := HttpErrBytes(404, errors.New("container not found"))
		w.Write(errBytes)
		return
	}

	_, _ = con, container
	// done := make(chan struct{})
	// sets := container.Streams.Metrics.Stream(done)
	// WriteSets(con, sets, done)

}
func (api *API) logsWS(w http.ResponseWriter, r *http.Request, id string) {
	con, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		w.Write(errBytes)
		return
	}

	container, exists := api.Controller.ContainerGet(id)
	if !exists {
		errBytes, _ := HttpErrBytes(404, errors.New("container not found"))
		w.Write(errBytes)
		return
	}
	_, _ = con, container

	// done := make(chan struct{})
	// sets := container.Streams.Logs.Stream(done)
	// WriteSets(con, sets, done)
}

func (api *API) Test(ctx *gin.Context) {
	api.testWs(ctx.Writer, ctx.Request)
}

func (api *API) testWs(w http.ResponseWriter, r *http.Request) {
	con, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		w.Write(errBytes)
		return
	}

	//done := make(chan struct{})
	client := api.Hub.CreateClient(con)
	client.Handle()
}
