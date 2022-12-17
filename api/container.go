package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KeepAliveMsg struct {
	KeepAlive bool `json:"keep_alive"`
}

func (api *API) Container(ctx *gin.Context) {
	id := ctx.Param("id")
	cont, exists := api.Controller.Container(id)
	if !exists {
		HttpErr(ctx, http.StatusNotFound, errors.New("container not found"))
	}
	ctx.JSON(http.StatusOK, cont)
}

func (api *API) Containers(ctx *gin.Context) {
	containers, err := api.Controller.Storage.Containers()
	if err != nil {
		HttpErr(ctx, http.StatusInternalServerError, err)
	}
	ctx.JSON(http.StatusOK, containers)
}

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

	result := api.Controller.Storage.DB.Metrics(id, tminP, tmaxP)
	// fmt.Println(result[id])

	ctx.JSON(http.StatusOK, result[id])
}

func (api *API) Stream(ctx *gin.Context) {
	api.StreamWS(ctx.Writer, ctx.Request)
}

func (api *API) StreamWS(w http.ResponseWriter, r *http.Request) {
	con, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		errBytes, _ := HttpErrBytes(500, err)
		w.Write(errBytes)
		return
	}

	client := api.Hub.CreateClient(con)
	client.Run()
}

func (api *API) Images(ctx *gin.Context) {
	images, err := api.Controller.Storage.Images()
	if err != nil {
		HttpErr(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, images)
}
