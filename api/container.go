package api

import (
	"encoding/json"
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
	} else {
		contJson, err := container.MarshalJSON()
		if err != nil {
			HttpErr(ctx, http.StatusInternalServerError, err)
			return
		}
		ctx.String(http.StatusOK, string(contJson))

	}
}

func (api *API) Containers(ctx *gin.Context) {
	b, err := api.Controller.Containers.MarshalJSON()
	if err != nil {
		HttpErr(ctx, http.StatusInternalServerError, errors.New("failed to fetch containers"))
		return
	}
	ctx.String(http.StatusOK, string(b))
}

func (api *API) streamMetrics(w http.ResponseWriter, r *http.Request, id string) {
	con, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		w.Write(errBytes)
	}

	container := api.Controller.Container(id)
	if container == (&dock.Container{}) {
		errBytes, _ := HttpErrBytes(404, errors.New("container not found"))
		w.Write(errBytes)
		return
	}

	done := make(chan bool)
	metrics := container.MetricsStream(done)

	for set := range metrics {
		setJson, err := json.Marshal(set)
		if err != nil {
			HttpErrBytes(0, err)
			con.Close()
			return
		}
		_ = con.WriteMessage(1, setJson)
	}

}

func (api *API) ContainerMetrics(ctx *gin.Context) {
	id := ctx.Param("id")
	api.streamMetrics(ctx.Writer, ctx.Request, id)
}
