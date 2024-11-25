package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	client *redis.Client
}

func New(redisURL string) (*Cache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error connecting to Redis: %v", err)
	}

	return &Cache{client: client}, nil
}

// Set stores a value in the cache with an expiration time
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from the cache
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *Cache) GetArticle(title string) (string, error) {
	ctx := context.Background()

	// Try to get from cache
	val, err := c.client.Get(ctx, title).Result()
	if err == redis.Nil {
		// Not in cache, fetch from Wikipedia and store
		article, err := fetchFromWikipedia(title)
		if err != nil {
			return "", fmt.Errorf("error fetching from Wikipedia: %v", err)
		}

		// Store in cache for 24 hours
		err = c.client.Set(ctx, title, article, 24*time.Hour).Err()
		if err != nil {
			return "", fmt.Errorf("error caching article: %v", err)
		}

		return article, nil
	} else if err != nil {
		return "", fmt.Errorf("error accessing cache: %v", err)
	}

	return val, nil
}

func fetchFromWikipedia(title string) (string, error) {
	endpoint := "https://en.wikipedia.org/w/api.php"
	params := url.Values{}
	params.Add("action", "parse")
	params.Add("page", title)
	params.Add("format", "json")
	params.Add("prop", "text")
	params.Add("redirects", "1")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?%s", endpoint, params.Encode()), nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Add user agent as Wikipedia API recommends
	req.Header.Set("User-Agent", "CodeSwitchAI/1.0 (https://github.com/mrconter1/codeswitch-ai)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Wikipedia API returned status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var result struct {
		Parse struct {
			Text map[string]string `json:"text"`
		} `json:"parse"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing JSON response: %v", err)
	}

	htmlContent, exists := result.Parse.Text["*"]
	if !exists {
		return "", fmt.Errorf("no content found in Wikipedia response")
	}

	log.Printf("Successfully fetched article: %s", title)
	return htmlContent, nil
}

// GetLanguages returns the list of supported languages
func (c *Cache) GetLanguages() []string {
	// For now, just return our supported languages
	return []string{"en", "sv"}
}
