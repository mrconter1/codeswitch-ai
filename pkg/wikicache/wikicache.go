package wikicache

import (
	"fmt"
	"io/ioutil"
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
		return content, nil
	}

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

	return string(body), nil
}
