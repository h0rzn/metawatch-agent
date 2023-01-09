package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// /container/:id endpoint for fetching single container by id
func (api *API) Container(ctx *gin.Context) {
	id := ctx.Param("id")
	if container, exists := api.Controller.Containers.Container(id); exists {
		ctx.JSON(http.StatusOK, container)
	} else {
		HttpErr(ctx, http.StatusNotFound, errors.New("container not found"))
	}
}

// /containers/all endpoint for fetching all containers
func (api *API) Containers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, api.Controller.Containers)
}

// /container/:id/metrics?from=X&to=Y endpoint for fetching container metrics
// between X and Y
func (api *API) Metrics(ctx *gin.Context) {
	id := ctx.Param("id")
	query := ctx.Request.URL.Query()

	if query.Get("from") == "" || query.Get("to") == "" {
		HttpErr(ctx, http.StatusBadRequest, errors.New("from=x and to=x required"))
		return
	}

	// 2022-12-15T13:00:00Z
	layout := time.RFC3339Nano
	tmin, err := time.Parse(layout, query.Get("from"))
	if err != nil || tmin.IsZero() {
		HttpErr(ctx, http.StatusBadRequest, err)
		return
	}

	tmax, err := time.Parse(layout, query.Get("to"))
	if err != nil || tmax.IsZero() {
		HttpErr(ctx, http.StatusBadRequest, err)
		return
	}

	tminP := primitive.NewDateTimeFromTime(tmin)
	tmaxP := primitive.NewDateTimeFromTime(tmax)

	result := api.Controller.DB.Metrics(id, tminP, tmaxP)
	// fmt.Println(result[id])

	ctx.JSON(http.StatusOK, result[id])
}

// /stream endpoint for accessing the websocket that supplies
// live metrics, logs and events
func (api *API) Stream(ctx *gin.Context) {
	con, err := upgrade.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		ctx.Writer.Write(errBytes)
		return
	}
	client := api.Hub.CreateClient(con)
	client.Run()
}


