package processor

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mrconter1/codeswitch-ai/pkg/claude"
)

type WordFrequency struct {
	word  string
	count int
}

type Processor struct {
	claudeClient    *claude.Client
	rateLimiter     <-chan time.Time
	enWordFreqs     []WordFrequency
	svWordFreqs     []WordFrequency
	frequencyLoader sync.Once
}

func New(claudeClient *claude.Client) *Processor {
	return &Processor{
		claudeClient: claudeClient,
		rateLimiter:  time.Tick(time.Second),
	}
}

// loadFrequencyData loads word frequency data from GitHub
func (p *Processor) loadFrequencyData() error {
	p.frequencyLoader.Do(func() {
		log.Println("Loading frequency dictionaries...")

		// Load English frequencies
		enResp, err := http.Get("https://raw.githubusercontent.com/hermitdave/FrequencyWords/master/content/2018/en/en_50k.txt")
		if err != nil {
			log.Printf("Error loading English frequencies: %v", err)
			return
		}
		defer enResp.Body.Close()

		scanner := bufio.NewScanner(enResp.Body)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) == 2 {
				count := 0
				fmt.Sscanf(parts[1], "%d", &count)
				p.enWordFreqs = append(p.enWordFreqs, WordFrequency{
					word:  parts[0],
					count: count,
				})
			}
		}

		// Load Swedish frequencies
		svResp, err := http.Get("https://raw.githubusercontent.com/hermitdave/FrequencyWords/master/content/2018/sv/sv_50k.txt")
		if err != nil {
			log.Printf("Error loading Swedish frequencies: %v", err)
			return
		}
		defer svResp.Body.Close()

		scanner = bufio.NewScanner(svResp.Body)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) == 2 {
				count := 0
				fmt.Sscanf(parts[1], "%d", &count)
				p.svWordFreqs = append(p.svWordFreqs, WordFrequency{
					word:  parts[0],
					count: count,
				})
			}
		}

		log.Printf("Loaded %d English words and %d Swedish words", len(p.enWordFreqs), len(p.svWordFreqs))
	})

	return nil
}

// calculateWordsNeeded uses Zipf's law to estimate how many top frequency words
// we need to translate to achieve the desired percentage
func (p *Processor) calculateWordsNeeded(percentage float64) int {
	// Simple Zipf's law implementation
	// In a natural language, frequency of nth most common word is proportional to 1/n
	totalWords := len(p.enWordFreqs)
	targetPercentage := percentage / 100.0

	// Calculate cumulative frequencies
	total := 0.0
	for i := 0; i < totalWords; i++ {
		total += 1.0 / float64(i+1)
	}

	// Find how many words we need
	cumulative := 0.0
	for i := 0; i < totalWords; i++ {
		cumulative += 1.0 / float64(i+1)
		if cumulative/total >= targetPercentage {
			return i + 1
		}
	}

	return totalWords
}

// findWordsToTranslate identifies which high-frequency words appear in the text
func (p *Processor) findWordsToTranslate(text string, numWords int) []string {
	words := strings.Fields(strings.ToLower(text))
	freqWordSet := make(map[string]bool)

	// Take top N frequency words
	for i := 0; i < numWords && i < len(p.enWordFreqs); i++ {
		freqWordSet[p.enWordFreqs[i].word] = true
	}

	// Find matches in text
	matches := make(map[string]bool)
	for _, word := range words {
		if freqWordSet[word] {
			matches[word] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(matches))
	for word := range matches {
		result = append(result, word)
	}

	return result
}

func (p *Processor) ProcessParagraph(content, sourceLang, targetLang string, percentage float64) (string, error) {
	// Ensure frequency data is loaded
	if err := p.loadFrequencyData(); err != nil {
		return "", fmt.Errorf("failed to load frequency data: %v", err)
	}

	// Log processing start
	log.Printf("Processing paragraph (%.0f%% target): %s...", percentage, content[:int(math.Min(float64(len(content)), 50))])

	// Calculate number of words needed based on Zipf's law
	wordsNeeded := p.calculateWordsNeeded(percentage)
	log.Printf("Calculated need for %d top-frequency words to achieve %.0f%%", wordsNeeded, percentage)

	// Find actual words to translate
	wordsToTranslate := p.findWordsToTranslate(content, wordsNeeded)
	log.Printf("Found %d matching high-frequency words in text: %v", len(wordsToTranslate), wordsToTranslate)

	// Wait for rate limiter
	<-p.rateLimiter

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create prompt for Claude
	prompt := fmt.Sprintf(`Given this paragraph in %s:

%s

Please translate ONLY these specific words to %s, keeping their exact position and context in the sentence. 
Words to translate: %v

Keep all other words unchanged. Maintain the original format, spacing, and punctuation.`,
		sourceLang, content, targetLang, wordsToTranslate)

	log.Printf("Sending request to Claude for code-switching")
	result, err := p.claudeClient.Complete(ctx, prompt)
	if err != nil {
		log.Printf("Error from Claude: %v", err)
		return "", err
	}

	log.Printf("Successfully processed paragraph: %s...", result[:int(math.Min(float64(len(result)), 50))])
	return result, nil
}
