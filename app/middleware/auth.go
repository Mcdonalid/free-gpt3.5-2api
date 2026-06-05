package middleware

import (
	"chat2api/app/common"
	"chat2api/app/conf"
	"strings"

	"github.com/gin-gonic/gin"
)

func V1Auth(c *gin.Context) {
	authToken := c.Request.Header.Get("Authorization")
	localToken := strings.TrimSpace(strings.TrimPrefix(authToken, "Bearer "))
	if strings.HasPrefix(localToken, "at-") {
		c.Next()
		return
	}
	if strings.HasPrefix(authToken, "Bearer eyJhbGciOiJSUzI1NiI") {
		c.Next()
		return
	}
	if authToken == "" && len(conf.App.Auth.AccessTokens) > 0 {
		common.ErrorResponse(c, 401, "You didn't provide an API key. You need to provide your API key in an Authorization header using Bearer auth (i.e. Authorization: Bearer YOUR_KEY)", nil)
		return
	}
	if !common.IsStrInArray(localToken, conf.App.Auth.AccessTokens) {
		common.ErrorResponse(c, 401, "Incorrect API key provided: sk-4yNZz***************************************6mjw.", nil)
		return
	}
	c.Next()
}
