package routers

import (
	"github.com/The-Promised-Neverland/master-server/internal/api/handlers"
	middleware "github.com/The-Promised-Neverland/master-server/internal/api/middlware"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/gin-gonic/gin"
)

type Router struct {
	WSHub      *ws.WSHub
	SSEHub     *sse.SSEHub
	Handler    *handlers.Handler
	WSHandler  *handlers.WebSocketHandler
	SSEHandler *handlers.SSEHandler
}

func NewRouter(wshub *ws.WSHub, sseHub *sse.SSEHub, handler *handlers.Handler, wsh *handlers.WebSocketHandler, sseH *handlers.SSEHandler) *Router {
	return &Router{
		WSHub:      wshub,
		SSEHub:     sseHub,
		Handler:    handler,
		WSHandler:  wsh,
		SSEHandler: sseH,
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
			agents.GET("", rtr.Handler.ListAgents)                      // list all agents
			agents.GET("/:id", rtr.Handler.GetAgent)                    // get agent data (last seen, isOnline, downtime)
			agents.GET("/:id/metrics", rtr.Handler.TriggerAgentMetrics) // get agent metrics
			agents.POST("/:id/restart", rtr.Handler.RestartAgent)       // restart a agent
			agents.POST("/:id/uninstall", rtr.Handler.UninstallAgent)   // uninstall a agent
		}
	}
	router.GET("/ws", rtr.WSHandler.UpgradeHandler)
	router.GET("/sse", rtr.SSEHandler.StreamHandler)

	return router
}
