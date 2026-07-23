package utils

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"sirtom/server/logger"
)

func ParseArrayQueryParam(s string) []string {
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		logger.StdErr.Panicln(err)
	}
	arr := strings.Split(decoded, ",")
	return arr
}

// Returns origin of the given request (i.e. http://localhost:8080 or http://localhost:3002 or https://gathering.sirthomasfoolery.com)
func GetOrigin(c *gin.Context) string {
	return c.Request.Header.Get("Origin")
}
