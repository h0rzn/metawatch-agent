package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/h0rzn/monitoring_agent/dock"
)

type KeepAliveMsg struct {
	KeepAlive bool `json:"keep_alive"`
}

func (api *API) Container(ctx *gin.Context) {
	id := ctx.Param("id")
	container := api.Controller.Container(id)
	if container == (&dock.Container{}) {
		HttpErr(ctx, http.StatusNotFound, errors.New("container not found"))
	}
	json, err := container.MarshalJSON()
	if err != nil {
		HttpErr(ctx, http.StatusNotFound, errors.New("failed to marshal container"))
	}
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", json)
}

func (api *API) Containers(ctx *gin.Context) {
	b, err := api.Controller.Containers.MarshalJSON()
	if err != nil {
		HttpErr(ctx, http.StatusInternalServerError, errors.New("failed to fetch containers"))
		return
	}
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", b)
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

	container := api.Controller.Container(id)
	if container == (&dock.Container{}) {
		errBytes, _ := HttpErrBytes(404, errors.New("container not found"))
		w.Write(errBytes)
		return
	}

	done := make(chan struct{})
	sets := container.Metrics.Stream(done)
	WriteSets(con, sets, done)

}
func (api *API) logsWS(w http.ResponseWriter, r *http.Request, id string) {
	con, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		w.Write(errBytes)
		return
	}

	container := api.Controller.Container(id)
	if container == (&dock.Container{}) {
		errBytes, _ := HttpErrBytes(404, errors.New("container not found"))
		w.Write(errBytes)
		return
	}

	done := make(chan struct{})
	sets := container.Logs.Stream(done)
	WriteSets(con, sets, done)
}
