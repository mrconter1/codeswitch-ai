package main

import (
	"log"
	"net/http"
	"os"

	"github.com/mrconter1/codeswitch-ai/internal/gateway"
	"github.com/mrconter1/codeswitch-ai/internal/processor"
	"github.com/mrconter1/codeswitch-ai/pkg/cache"
	"github.com/mrconter1/codeswitch-ai/pkg/claude"
)

func main() {
	// Initialize Claude client
	claudeClient := claude.New(os.Getenv("CLAUDE_API_KEY"))

	// Initialize cache
	cache, err := cache.New(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Initialize processor
	processor := processor.New(claudeClient)

	// Initialize gateway
	gateway := gateway.New(cache, processor)

	// Setup routes
	http.HandleFunc("/codeswitch", gateway.HandleCodeSwitch)

	// Start server
	log.Printf("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
