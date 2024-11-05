package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/mrconter1/codeswitch-ai/pkg/claude"
)

type Processor struct {
	claudeClient *claude.Client
	rateLimiter  <-chan time.Time
}

func New(claudeClient *claude.Client) *Processor {
	// Start with a simple rate limiter - 1 request per second
	return &Processor{
		claudeClient: claudeClient,
		rateLimiter:  time.Tick(time.Second),
	}
}

func (p *Processor) ProcessParagraph(content, sourceLang, targetLang string, percentage float64) (string, error) {
	// Wait for rate limiter
	<-p.rateLimiter

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create prompt for Claude
	prompt := fmt.Sprintf(`Given this paragraph in %s, rewrite it so that %.1f%% of the words are kept in %s while
the rest are translated to %s. Focus on keeping the most important words in the original language:

%s`, sourceLang, percentage, sourceLang, targetLang, content)

	// Process with Claude
	return p.claudeClient.Complete(ctx, prompt)
}
