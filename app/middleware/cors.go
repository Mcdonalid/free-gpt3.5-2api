package middleware

import "github.com/gin-gonic/gin"

func V1Cors(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Token, Content-Type, Accept")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}
	c.Next()
}
