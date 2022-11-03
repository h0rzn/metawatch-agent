package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/h0rzn/monitoring_agent/dock"
)

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
	con, _ := upgrade.Upgrade(w, r, nil) // handle error

	container := api.Controller.Container(id)
	if container == (&dock.Container{}) {
		fmt.Println("container not found") // handle error
	}

	done := make(chan bool)
	metrics := container.MetricsStream(done)

	for set := range metrics {
		setJson, err := json.Marshal(set)
		fmt.Println("sending:", string(setJson))
		if err != nil {
			fmt.Println("error writing json string to ws con")
		}
		_ = con.WriteMessage(1, setJson)
		// implement timeout, 30s?
	}

}

func (api *API) ContainerMetrics(ctx *gin.Context) {
	id := ctx.Param("id")
	api.streamMetrics(ctx.Writer, ctx.Request, id)
}
