package wikicache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
)

// ArticleCache is a thread-safe in-memory cache
type ArticleCache struct {
	cache map[string]string
	mutex sync.RWMutex
}

// NewArticleCache creates a new ArticleCache
func NewArticleCache() *ArticleCache {
	return &ArticleCache{
		cache: make(map[string]string),
	}
}

// GetArticle returns the cached article or fetches it if not present
func (ac *ArticleCache) GetArticle(title string) (string, error) {
	// Check if article is in cache
	ac.mutex.RLock()
	content, found := ac.cache[title]
	ac.mutex.RUnlock()

	if found {
		log.Printf("Cache hit for title: %s\n", title)
		return content, nil
	}

	log.Printf("Cache miss for title: %s. Fetching from Wikipedia...\n", title)

	// Fetch article since it's not in cache
	content, err := fetchArticleFromWikipedia(title)
	if err != nil {
		return "", err
	}

	// Store the fetched article in cache
	ac.mutex.Lock()
	ac.cache[title] = content
	ac.mutex.Unlock()

	return content, nil
}

// fetchArticleFromWikipedia fetches the article HTML from Wikipedia
func fetchArticleFromWikipedia(title string) (string, error) {
	endpoint := "https://en.wikipedia.org/w/api.php"
	params := url.Values{}
	params.Add("action", "parse")
	params.Add("page", title)
	params.Add("format", "json")
	params.Add("prop", "text")
	params.Add("redirects", "1")

	resp, err := http.Get(fmt.Sprintf("%s?%s", endpoint, params.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Navigate the JSON to extract the HTML content
	parse, ok := result["parse"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format: missing 'parse'")
	}

	text, ok := parse["text"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format: missing 'text'")
	}

	htmlContent, ok := text["*"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format: missing '*'")
	}

	return htmlContent, nil
}
