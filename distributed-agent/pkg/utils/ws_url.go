package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func BuildWebSocketURL(baseURL, agentID string, name string, os string) string {
	wsURL := strings.Replace(baseURL, "https", "wss", 1)
	wsURL = strings.Replace(wsURL, "http", "ws", 1)
	return fmt.Sprintf("%s/ws?name=%s&id=%s&os=%s", wsURL, url.QueryEscape(name), url.QueryEscape(agentID), url.QueryEscape(os))
}
