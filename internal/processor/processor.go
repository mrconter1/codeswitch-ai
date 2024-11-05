package processor

import (
	"bufio"
	"context"
	"fmt"
	"log"
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

		log.Printf("Loaded %d English words and %d Swedish words",
			len(p.enWordFreqs), len(p.svWordFreqs))
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

func (p *Processor) createCodeSwitchPrompt(content string, originalWords []string, sourceLang, targetLang string) string {
	// Create a bullet list of words with their contexts
	var wordContexts []string
	for _, word := range originalWords {
		// Find a few words before and after for context
		index := strings.Index(strings.ToLower(content), strings.ToLower(word))
		if index >= 0 {
			start := max(0, index-20)
			end := min(len(content), index+len(word)+20)
			context := content[start:end]
			wordContexts = append(wordContexts, fmt.Sprintf("• %q appears in: %q", word, context))
		}
	}

	prompt := fmt.Sprintf(`You are a skilled linguistic expert in code-switching between %s and %s. 
Please create a naturally code-switched version of this text by translating ONLY the specified words from %s to %s.

Original paragraph:
%s

Words to translate (with their contexts):
%s

Instructions:
1. ONLY translate the listed words to %s
2. Keep all other words in their original %s form
3. Ensure grammatical agreement between the languages
4. Maintain all original formatting, punctuation, and capitalization
5. The translation should feel natural and maintain readability
6. Adapt articles and word forms to fit the grammar of both languages

Example code-switching:
English: "The cat was sleeping on the table"
Words to switch: [was, on, the]
Result: "The cat var sleeping på table"

Please provide ONLY the code-switched paragraph as output, without explanations.`,
		sourceLang, targetLang,
		sourceLang, targetLang,
		content,
		strings.Join(wordContexts, "\n"),
		targetLang,
		sourceLang)

	log.Printf("Created prompt for %d words. First few words to translate: %v",
		len(originalWords),
		originalWords[:min(5, len(originalWords))])

	return prompt
}

func (p *Processor) ProcessParagraph(content, sourceLang, targetLang string, percentage float64) (string, error) {
	// Ensure frequency data is loaded
	if err := p.loadFrequencyData(); err != nil {
		return "", fmt.Errorf("failed to load frequency data: %v", err)
	}

	log.Printf("Processing paragraph (%.0f%% target): %s...",
		percentage,
		content[:min(50, len(content))])

	// Calculate number of words needed based on Zipf's law
	wordsNeeded := p.calculateWordsNeeded(percentage)
	log.Printf("Calculated need for %d top-frequency words to achieve %.0f%%",
		wordsNeeded,
		percentage)

	// Find actual words to translate
	wordsToTranslate := p.findWordsToTranslate(content, wordsNeeded)
	log.Printf("Found %d matching high-frequency words in text: %v",
		len(wordsToTranslate),
		wordsToTranslate)

	// Wait for rate limiter
	<-p.rateLimiter

	// Create prompt
	prompt := p.createCodeSwitchPrompt(content, wordsToTranslate, sourceLang, targetLang)

	log.Printf("Sending request to Claude for code-switching")
	result, err := p.claudeClient.Complete(context.Background(), prompt)
	if err != nil {
		return "", fmt.Errorf("error from Claude: %v", err)
	}

	// Log a preview of the result
	log.Printf("Successfully processed paragraph: %s...",
		result[:min(50, len(result))])

	return result, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function for max
func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
