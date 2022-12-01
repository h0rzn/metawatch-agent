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
