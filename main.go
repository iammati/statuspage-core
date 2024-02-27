package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"infraops.dev/statuspage-core/config"
	"infraops.dev/statuspage-core/handlers"
	"infraops.dev/statuspage-core/websocket"
)

func main() {
	// Bootstrapping the application
	config.Bootstrap()

	port := "8080"
	srv := &http.Server{
		Addr: ":" + port,
	}

	// HTTP server
	http.HandleFunc("/up", handlers.HandleUp)
	http.HandleFunc("/certinfo", handlers.HandleCertInfo)

	// WebSocket server
	http.HandleFunc("/ws", websocket.Handle)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			PrintLog("listen: ", true)
			log.Fatalf("%s\n", err)
		}
	}()
	PrintLog(fmt.Sprintf("HTTP server listening on port *:%s...", port))

	go handlers.CleanupInactiveHosts(5 * time.Second)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	PrintLog("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		PrintLog("Server forced to shutdown: ", true)
		log.Fatalf("%s\n", err)
	}

	PrintLog("Server exiting")
}

func PrintLog(reason string, doNotLogIt ...bool) {
	handlers.LogUpdatetimeEvent(handlers.UpdatetimeEvent{
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Reason:    reason,
	})

	if len(doNotLogIt) == 0 {
		log.Println(reason)
	}
}
