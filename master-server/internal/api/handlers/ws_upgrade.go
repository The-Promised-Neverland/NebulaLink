package handlers

import (
	"fmt"
	"net/http"

	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	Hub *ws.WSHub
}

func NewWebSocketHandler(hub *ws.WSHub) *WebSocketHandler {
	return &WebSocketHandler{Hub: hub}
}

func (wsh *WebSocketHandler) UpgradeHandler(c *gin.Context) {
	name := c.Query("name")
	id := c.Query("id")
	os := c.Query("os")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade WebSocket: %v\n", err)
		return
	}
	fmt.Printf("New connection -> ID: %s, Name: %s, OS: %s\n", id, name, os)
	wsh.Hub.Connect(name, id, os, conn)
}
