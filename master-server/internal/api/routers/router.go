package routers

import (
	"github.com/The-Promised-Neverland/master-server/internal/api/handlers"
	middleware "github.com/The-Promised-Neverland/master-server/internal/api/middlware"
	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/gin-gonic/gin"
)

type Router struct {
	Hub       *ws.Hub
	Handler   *handlers.Handler
	WSHandler *handlers.WebSocketHandler
}

func NewRouter(hub *ws.Hub, handler *handlers.Handler, wsh *handlers.WebSocketHandler) *Router {
	return &Router{
		Hub:       hub,
		Handler:   handler,
		WSHandler: wsh,
	}
}

func (rtr *Router) SetupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CorsMiddleware())

	router.GET("/health", rtr.Handler.HealthCheck)

	v1 := router.Group("/api/v1")
	{
		agents := v1.Group("/agents")
		{
			agents.GET("", rtr.Handler.ListAgents)                    // list all agents
			agents.GET("/:id", rtr.Handler.GetAgent)                  // get agent data (last seen, isOnline, downtime)
			agents.GET("/:id/metrics", rtr.Handler.TriggerAgentMetrics)   // get agent metrics
			agents.POST("/:id/restart", rtr.Handler.RestartAgent)     // restart a agent
			agents.POST("/:id/uninstall", rtr.Handler.UninstallAgent) // uninstall a agent
		}

		tasks := v1.Group("/tasks")
		{
			tasks.POST("", func(ctx *gin.Context) {})    // post task
			tasks.GET("/:id", func(ctx *gin.Context) {}) // Get task status
			tasks.GET("", func(ctx *gin.Context) {})     // All tasks pending
		}
	}
	router.GET("/ws", rtr.WSHandler.UpgradeHandler) // Upgrade to websocket request

	return router
}
