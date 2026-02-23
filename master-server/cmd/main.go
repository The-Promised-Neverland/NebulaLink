package main

import (
	"log"
	"net/http"
	"os"

	"github.com/The-Promised-Neverland/master-server/internal/api/handlers"
	"github.com/The-Promised-Neverland/master-server/internal/api/routers"
	"github.com/The-Promised-Neverland/master-server/internal/service"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/The-Promised-Neverland/master-server/pkg/system"
)

func main() {
	system.InitStartTime()
	sseHub := sse.NewSSEHub()
	wsHub := ws.NewWSHub(sseHub)
	wsHub.RegisterDefaultHandlers()
	svc := service.NewService(wsHub, sseHub)
	handler := handlers.NewHandler(svc)
	wsHandler := handlers.NewWebSocketHandler(wsHub)
	sseHandler := handlers.NewSSEHandler(sseHub)
	sseHandler.SetService(svc)
	router := routers.NewRouter(wsHub, sseHub, handler, wsHandler, sseHandler).SetupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8430"
	}
	addr := ":" + port
	log.Printf("Server started at %s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
