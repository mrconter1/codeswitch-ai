package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mrconter1/codeswitch-ai/pkg/wikicache"
)

func main() {
	// Create a new article cache
	cache := wikicache.NewArticleCache()

	// Define the handler function
	http.HandleFunc("/article", func(w http.ResponseWriter, r *http.Request) {
		// Get the 'title' query parameter
		title := r.URL.Query().Get("title")
		if title == "" {
			http.Error(w, "Title parameter is missing", http.StatusBadRequest)
			return
		}

		// Get the article content
		content, err := cache.GetArticle(title)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching article: %v", err), http.StatusInternalServerError)
			return
		}

		// Set content type to JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(content))
	})

	// Start the server
	fmt.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
