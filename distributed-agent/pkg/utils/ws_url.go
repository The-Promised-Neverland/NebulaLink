package utils

import (
	"fmt"
	"strings"
)

func BuildWebSocketURL(baseURL, agentID string) string {
	wsURL := strings.Replace(baseURL, "https", "wss", 1)
	wsURL = strings.Replace(wsURL, "http", "ws", 1)
	return fmt.Sprintf("%s/ws?role=agent:%s", wsURL, agentID)
}
