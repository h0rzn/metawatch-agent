package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// /about endpoint for general data like docker (api) verion, ...
func (a *API) About(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, a.Controller.About)
}