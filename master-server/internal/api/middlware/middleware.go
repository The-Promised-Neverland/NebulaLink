package middleware

import (
	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func AgentExistenceMiddleware(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentID := c.Param("id")
		if agentID == "" {
			c.Next() 
			return
		}
		hub.Mutex.RLock()
		_, exists := hub.Connections[agentID]
		hub.Mutex.RUnlock()
		if !exists {
			c.JSON(404, gin.H{
				"success": false,
				"error":   "Agent ID not found",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
