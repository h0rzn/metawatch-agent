package api

import (
	"errors"
	"fmt"
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
	json := cont.JSONSkel()
	ctx.JSON(http.StatusOK, json)
}

func (api *API) Containers(ctx *gin.Context) {
	json := api.Controller.Storage.JSONSkel()
	ctx.JSON(http.StatusOK, json)
}

func (api *API) Metrics(ctx *gin.Context) {
	id := ctx.Param("id")

	query := ctx.Request.URL.Query()
	layout := "2006-01-02T14:13:00"
	tmin, err := time.Parse(layout, query.Get("from"))
	if err != nil {
		fmt.Println("[API] time parse", err)
		return
	}
	// tmin.IsZero()
	tminPrim := primitive.NewDateTimeFromTime(tmin)

	tmax := primitive.NewDateTimeFromTime(time.Now())
	fmt.Println("PARAMS", id, tminPrim, tmax)

	result := api.Controller.Storage.DB.Metrics(id, primitive.NewDateTimeFromTime(tmin), tmax)
	_ = result
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

	//done := make(chan struct{})
	client := api.Hub.CreateClient(con)
	client.Handle()
}
