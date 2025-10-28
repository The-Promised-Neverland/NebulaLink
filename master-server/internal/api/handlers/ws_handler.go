package handlers

import (
	"fmt"
	"net/http"

	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	Hub *ws.Hub
}

func NewWebSocketHandler(hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{Hub: hub}
}

func (wsh *WebSocketHandler) UpgradeHandler(c *gin.Context) {
	agentID := c.Query("id")
	role := c.Query("role")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing agent ID"})
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("❌ Failed to upgrade WebSocket for agent %s: %v\n", agentID, err)
		return
	}
	switch role {
	case "frontend":
		wsh.Hub.RegisterFrontend(conn)
	default:
		if agentID == "" {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Missing agent ID"}`))
			conn.Close()
			return
		}
		wsh.Hub.RegisterConnection(agentID, conn)
	}
}
