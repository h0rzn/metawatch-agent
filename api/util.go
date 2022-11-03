package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func HttpErr(ctx *gin.Context, code int, err error) {
	errMap := make(map[string]interface{})
	errMap["code"] = code
	errMap["message"] = err.Error()
	ctx.JSON(code, errMap)
}

func HttpErrBytes(code int, err error) ([]byte, error) {
	errMap := map[string]interface{}{
		"code":    code,
		"message": err.Error(),
	}
	errBytes, err := json.Marshal(errMap)
	if err != nil {
		return []byte{}, err
	}
	return errBytes, nil
}
