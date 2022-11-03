package api

import "github.com/gin-gonic/gin"

func HttpErr(ctx *gin.Context, code int, err error) {
	errMap := make(map[string]interface{})
	errMap["code"] = code
	errMap["message"] = err.Error()
	ctx.JSON(code, errMap)
}
