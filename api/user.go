package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
)

type UserPatch struct {
}

func (api *API) GetUsers(ctx *gin.Context) {
	users, err := api.Controller.DB.GetUsers()
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, users)
}

func (api *API) RegisterUser(ctx *gin.Context) {
	var user db.User
	err := ctx.ShouldBindJSON(&user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"error": "malformed input"})
		return
	}
	err = api.Controller.DB.InsertUser(user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusAccepted, struct{}{})

}

func (api *API) PatchUser(ctx *gin.Context) {
	id := ctx.Param("id")

	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read json"})
		return
	}
	var jsonData map[string]string
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to parse"})
		return
	}

	result, err := api.Controller.DB.UpdateUser(jsonData, id)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusAccepted, result)

}

func (api *API) RemoveUser(ctx *gin.Context) {
	id := ctx.Param("id")
	err := api.Controller.DB.RemoveUser(id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusAccepted, struct{}{})
}
