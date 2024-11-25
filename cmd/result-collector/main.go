package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mrconter1/codeswitch-ai/pkg/cache"
	"github.com/mrconter1/codeswitch-ai/pkg/messagebroker"
)

type ResultCollector struct {
	cache    *cache.Cache
	rabbitmq *messagebroker.RabbitMQ
	results  sync.Map
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

	collector := &ResultCollector{
		cache:    cacheClient,
		rabbitmq: rabbitmq,
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

	// Start consuming results
	if err := collector.startCollecting(ctx); err != nil {
		log.Fatalf("Failed to start collecting: %v", err)
	}
}

func (rc *ResultCollector) startCollecting(ctx context.Context) error {
	tasks, err := rc.rabbitmq.ConsumeParagraphs(ctx)
	if err != nil {
		return err
	}

	log.Println("Result collector started, waiting for results...")

	for task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Store result in cache with expiration
			key := "result:" + task.ID
			rc.cache.Set(ctx, key, task.Text, 24*time.Hour)
			log.Printf("Collected result for task %s", task.ID)
		}
	}

	return nil
}
