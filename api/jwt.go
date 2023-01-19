package api

import (
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

const (
	jwtTimeout    = time.Hour
	jwtMaxRefresh = time.Hour
	jwtIDKey      = "id"
)

type JWTUser struct {
	Name string
}

type JWTLogin struct {
	Name   string `form:"username" json:"username" binding:"required"`
	Passwd string `form:"password" json:"password" binding:"required"`
}

func JWT() (*jwt.GinJWTMiddleware, error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Key:         []byte("jwt-key"),
		Timeout:     jwtTimeout,
		MaxRefresh:  jwtMaxRefresh,
		IdentityKey: jwtIDKey,

		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*JWTUser); ok {
				return jwt.MapClaims{
					jwt.IdentityKey: v.Name,
				}
			}
			return jwt.MapClaims{}
		},

		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &JWTUser{
				Name: claims[jwt.IdentityKey].(string),
			}
		},

		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals JWTLogin
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			userID := loginVals.Name
			password := loginVals.Passwd

			if (userID == "master" && password == "master") {
				return &JWTUser{
					Name: userID,
				}, nil
			}

			return nil, jwt.ErrFailedAuthentication
		},

		Authorizator: func(data interface{}, c *gin.Context) bool {
			if v, ok := data.(*JWTUser); ok && v.Name == "master" {
				return true
			}

			return false
		},

		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},

		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})
}
