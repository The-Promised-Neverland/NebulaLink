package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func BuildWebSocketURL(baseURL, agentID string, name string) string {
	wsURL := strings.Replace(baseURL, "https", "wss", 1)
	wsURL = strings.Replace(wsURL, "http", "ws", 1) // local testing. uncomment in prod
	return fmt.Sprintf("%s/ws?role=%s&name=%s", wsURL, url.QueryEscape("agent:"+agentID), url.QueryEscape(name))
}
