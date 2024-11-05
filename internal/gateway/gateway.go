package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

func (g *Gateway) HandleCodeSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.CodeSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Try to get article from cache
	article, err := g.cache.GetArticle(req.Title)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching article: %v", err), http.StatusInternalServerError)
		return
	}

	// Split into paragraphs (simple implementation to start)
	paragraphs := strings.Split(article, "</p>")

	// Process each paragraph
	processedParagraphs := make([]string, len(paragraphs))
	for i, p := range paragraphs {
		processed, err := g.processor.ProcessParagraph(p, req.SourceLanguage, req.TargetLanguage, req.SwitchPercent)
		if err != nil {
			// For now, just use original if processing fails
			processedParagraphs[i] = p
			continue
		}
		processedParagraphs[i] = processed
	}

	// Reassemble
	result := strings.Join(processedParagraphs, "</p>")

	response := api.CodeSwitchResponse{
		HTML:     result,
		Title:    req.Title,
		Language: req.TargetLanguage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
