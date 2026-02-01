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
	role := c.Query("role")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade WebSocket: %v\n", err)
		return
	}
	fmt.Print("role recieved -> ", role)
	wsh.Hub.Connect(role, conn)
}
