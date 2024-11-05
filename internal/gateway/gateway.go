package gateway

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mrconter1/codeswitch-ai/api"
	"github.com/mrconter1/codeswitch-ai/internal/processor"
	"github.com/mrconter1/codeswitch-ai/pkg/cache"
)

type Gateway struct {
	cache     *cache.Cache
	processor *processor.Processor
}

func New(cache *cache.Cache, processor *processor.Processor) *Gateway {
	return &Gateway{
		cache:     cache,
		processor: processor,
	}
}

func extractTextFromNode(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += extractTextFromNode(c)
	}
	return result
}

func findParagraphs(n *html.Node) []*html.Node {
	var paragraphs []*html.Node
	if n.Type == html.ElementNode && n.Data == "p" {
		paragraphs = append(paragraphs, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		paragraphs = append(paragraphs, findParagraphs(c)...)
	}
	return paragraphs
}

func (g *Gateway) HandleCodeSwitch(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("Received code-switching request")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.CodeSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Processing request for article '%s' (%s → %s, %.1f%%)",
		req.Title, req.SourceLanguage, req.TargetLanguage, req.SwitchPercent)

	// Get article from cache
	log.Printf("Fetching article from cache: %s", req.Title)
	article, err := g.cache.GetArticle(req.Title)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching article: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Retrieved article: %d bytes", len(article))

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(article))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing HTML: %v", err), http.StatusInternalServerError)
		return
	}

	// Find all paragraphs
	paragraphs := findParagraphs(doc)
	log.Printf("Found %d paragraphs to process", len(paragraphs))

	// Process each paragraph
	successCount := 0
	failCount := 0

	for i, p := range paragraphs {
		// Extract text content
		originalText := strings.TrimSpace(extractTextFromNode(p))
		if len(originalText) < 10 { // Skip very short paragraphs
			continue
		}

		log.Printf("Processing paragraph %d/%d (%d characters)", i+1, len(paragraphs), len(originalText))

		// Process the paragraph
		processed, err := g.processor.ProcessParagraph(originalText, req.SourceLanguage, req.TargetLanguage, req.SwitchPercent)
		if err != nil {
			log.Printf("Error processing paragraph %d: %v", i+1, err)
			failCount++
			continue
		}

		// Replace the original text with processed text
		p.FirstChild.Data = processed
		successCount++

		log.Printf("Successfully processed paragraph %d", i+1)
	}

	// Convert back to HTML string
	var b strings.Builder
	err = html.Render(&b, doc)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error rendering HTML: %v", err), http.StatusInternalServerError)
		return
	}

	response := api.CodeSwitchResponse{
		HTML:     b.String(),
		Title:    req.Title,
		Language: req.TargetLanguage,
	}

	log.Printf("Request completed in %.2fs (success: %d, failed: %d paragraphs)",
		time.Since(startTime).Seconds(), successCount, failCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
