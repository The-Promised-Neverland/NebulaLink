package handlers

import (
	"fmt"
	"net/http"
	"strings"

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
	roleParam := c.Query("role")
	name := c.Query("name")

	var agentID string

	// Agent connections send role in the form "agent:ID"
	if strings.HasPrefix(roleParam, "agent:") {
		parts := strings.SplitN(roleParam, ":", 2)
		if len(parts) == 2 {
			agentID = parts[1]
		} else {
			agentID = roleParam
		}
	} else {
		// Frontend (and any non-agent) connections use their role string as the ID
		agentID = roleParam
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade WebSocket: %v\n", err)
		return
	}
	fmt.Printf("New connection -> ID: %s, Name: %s, Role: %s\n", agentID, name, roleParam)

	// Store connection under its ID; name is kept separately on the connection
	wsh.Hub.Connect(name, agentID, conn)
}
