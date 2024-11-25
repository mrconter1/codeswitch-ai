package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
)

type FrequencyCalculator struct {
	redisClient *redis.Client
}

type CalculateRequest struct {
	Language   string  `json:"language"`
	Percentage float64 `json:"percentage"`
}

func NewFrequencyCalculator(redisURL string) (*FrequencyCalculator, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	return &FrequencyCalculator{redisClient: client}, nil
}

func (fc *FrequencyCalculator) handleCalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Calculate number of words needed based on Zipf's law
	words, err := fc.calculateWordsForPercentage(r.Context(), req.Language, req.Percentage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(words)
}

func (fc *FrequencyCalculator) calculateWordsForPercentage(ctx context.Context, lang string, percentage float64) ([]string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("freq:%s:%.2f", lang, percentage)
	cached, err := fc.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var words []string
		if err := json.Unmarshal([]byte(cached), &words); err == nil {
			return words, nil
		}
	}

	// Fetch full word list
	fullList, err := fc.getFullList(ctx, lang)
	if err != nil {
		return nil, err
	}

	// Calculate how many words we need using Zipf's law
	totalWords := len(fullList)
	targetPercentage := percentage / 100.0

	// Simple Zipf implementation
	total := 0.0
	for i := 0; i < totalWords; i++ {
		total += 1.0 / float64(i+1)
	}

	wordsNeeded := 0
	cumulative := 0.0
	for i := 0; i < totalWords; i++ {
		cumulative += 1.0 / float64(i+1)
		if cumulative/total >= targetPercentage {
			wordsNeeded = i + 1
			break
		}
	}

	result := fullList[:wordsNeeded]

	// Cache the result
	if cached, err := json.Marshal(result); err == nil {
		fc.redisClient.Set(ctx, cacheKey, cached, 24*time.Hour)
	}

	return result, nil
}

func (fc *FrequencyCalculator) getFullList(ctx context.Context, lang string) ([]string, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("wordlist:%s", lang)
	cached, err := fc.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var words []string
		if err := json.Unmarshal([]byte(cached), &words); err == nil {
			return words, nil
		}
	}

	// Fetch from GitHub if not in cache
	url := fmt.Sprintf("https://raw.githubusercontent.com/hermitdave/FrequencyWords/master/content/2018/%s/%s_50k.txt", lang, lang)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch word list: %v", err)
	}
	defer resp.Body.Close()

	var words []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 1 {
			words = append(words, parts[0])
		}
	}

	if len(words) == 0 {
		return nil, fmt.Errorf("no words found for language %s", lang)
	}

	// Cache the result
	if cached, err := json.Marshal(words); err == nil {
		fc.redisClient.Set(ctx, cacheKey, cached, 24*time.Hour)
	}

	return words, nil
}

func main() {
	calculator, err := NewFrequencyCalculator(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("Failed to initialize calculator: %v", err)
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

	// Setup HTTP server
	http.HandleFunc("/calculate", calculator.handleCalculate)

	server := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Println("Frequency calculator service started on :8080")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
