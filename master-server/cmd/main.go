package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/The-Promised-Neverland/master-server/internal/api/handlers"
	"github.com/The-Promised-Neverland/master-server/internal/api/routers"
	"github.com/The-Promised-Neverland/master-server/internal/service"
	"github.com/The-Promised-Neverland/master-server/internal/ws"
	"github.com/The-Promised-Neverland/master-server/pkg/system"
)

func main() {
	system.InitStartTime()
	hub := ws.NewHub(func(msgType string, payload any) {
		fmt.Println("Received message:", msgType, payload)
	})
	svc := service.NewService(hub)
	handler := handlers.NewHandler(svc)
	wsHandler := handlers.NewWebSocketHandler(hub)
	router := routers.NewRouter(hub, handler, wsHandler).SetupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Server started at %s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
