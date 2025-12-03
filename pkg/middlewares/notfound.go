package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func NotMatchRouteMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "notFound",
			"path": c.FullPath(),
		})
	}
}

func NotMatchMethodsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "notMethods",
		})
	}
}
