package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mrconter1/codeswitch-ai/api"
)

func main() {
	// Command line flags
	title := flag.String("title", "Marie-Louise_Damien", "Wikipedia article title")
	sourceLang := flag.String("source", "en", "Source language")
	targetLang := flag.String("target", "sv", "Target language")
	percentage := flag.Float64("percent", 50.0, "Percentage to code-switch")
	serverURL := flag.String("url", "http://localhost:8080", "CodeSwitch API server URL")
	flag.Parse()

	// Create the request
	req := api.CodeSwitchRequest{
		Title:          *title,
		SourceLanguage: *sourceLang,
		TargetLanguage: *targetLang,
		SwitchPercent:  *percentage,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("Error marshaling request: %v", err)
	}

	// Send request
	log.Printf("Sending request to process article '%s' (%s → %s, %.1f%%)",
		req.Title, req.SourceLanguage, req.TargetLanguage, req.SwitchPercent)

	resp, err := http.Post(fmt.Sprintf("%s/codeswitch", *serverURL),
		"application/json",
		bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	// Check if it's an error response
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %s", body)
	}

	// Parse response
	var result api.CodeSwitchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("Error parsing response: %v", err)
	}

	// Print result
	fmt.Printf("\nProcessed Article:\n%s\n", result.HTML)
}
