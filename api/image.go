package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// /image/:id endpoint for fetching single image by id
func (api *API) Image(ctx *gin.Context) {
	id := ctx.Param("id")
	if img, exists := api.Controller.Storage.ImageStore.Image(id); exists {
		ctx.JSON(http.StatusOK, img)
	} else {
		HttpErr(ctx, http.StatusNotFound, errors.New("image not found"))
	}
}

// /images endpoint to fetch all images
func (api *API) Images(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, api.Controller.Storage.ImageStore)
}
