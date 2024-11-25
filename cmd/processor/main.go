package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrconter1/codeswitch-ai/pkg/claude"
	"github.com/mrconter1/codeswitch-ai/pkg/messagebroker"
)

func main() {
	// Initialize components
	claudeClient := claude.New(os.Getenv("CLAUDE_API_KEY"))

	rabbitmq, err := messagebroker.NewRabbitMQ(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
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

	// Start consuming tasks
	tasks, err := rabbitmq.ConsumeParagraphs(ctx)
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	log.Println("Processor started, waiting for tasks...")

	// Process tasks until context is cancelled
	for task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
			processedText, err := claudeClient.Complete(ctx, createPrompt(task))
			if err != nil {
				log.Printf("Error processing task %s: %v", task.ID, err)
				continue
			}

			// Publish result back to queue
			resultTask := messagebroker.ParagraphTask{
				ID:         task.ID,
				Text:       processedText,
				SourceLang: task.SourceLang,
				TargetLang: task.TargetLang,
			}

			if err := rabbitmq.PublishParagraph(ctx, resultTask); err != nil {
				log.Printf("Error publishing result for task %s: %v", task.ID, err)
				continue
			}

			log.Printf("Successfully processed task %s", task.ID)
		}
	}
}

func createPrompt(task messagebroker.ParagraphTask) string {
	return fmt.Sprintf(`Translate the following words from %s to %s in this text, maintaining their context and grammar:

Text: %s

Words to translate: %v

Please return only the processed text with the translations.`,
		task.SourceLang,
		task.TargetLang,
		task.Text,
		task.Words)
}
