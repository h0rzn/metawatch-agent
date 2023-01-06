package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (api *API) Images(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, api.Controller.Storage.ImageStore)
}
