package api

// CodeSwitchRequest represents the incoming request for code-switching
type CodeSwitchRequest struct {
	Title          string  `json:"title"`
	SourceLanguage string  `json:"sourceLang"`
	TargetLanguage string  `json:"targetLang"`
	SwitchPercent  float64 `json:"percentage"`
}

// CodeSwitchResponse represents the response with the processed article
type CodeSwitchResponse struct {
	HTML     string `json:"html"`
	Title    string `json:"title"`
	Language string `json:"language"`
}

// Error response for when things go wrong
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
