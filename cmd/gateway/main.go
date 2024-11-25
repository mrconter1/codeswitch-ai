package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrconter1/codeswitch-ai/pkg/cache"
	"github.com/mrconter1/codeswitch-ai/pkg/messagebroker"
)

type Gateway struct {
	cache    *cache.Cache
	rabbitmq *messagebroker.RabbitMQ
}

func main() {
	// Initialize components
	cacheClient, err := cache.New(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	rabbitmq, err := messagebroker.NewRabbitMQ(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	gateway := &Gateway{
		cache:    cacheClient,
		rabbitmq: rabbitmq,
	}

	// Setup HTTP server
	http.HandleFunc("/codeswitch", gateway.handleCodeSwitch)

	server := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	// Create context that listens for signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		cancel()
	}()

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Println("Gateway service started on :8080")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}

func (g *Gateway) handleCodeSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use the cache for article fetching
	// Implementation here...
}
