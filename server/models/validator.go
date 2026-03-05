package models

// ValidateRequest represents the incoming request with URLs to validate
type ValidateRequest struct {
	URLs []string `json:"urls"`
}

// ValidationResult represents the validation result for a single URL
// Status can be: "Valid", "Invalid", or "Blocked by Server"
type ValidationResult struct {
	URL    string `json:"url"`
	Status string `json:"status"` // Simplified: "Valid", "Invalid", "Blocked by Server"
	Index  int    `json:"index"`
}

// ValidateResponse represents the complete response with all validation results
type ValidateResponse struct {
	Results []ValidationResult `json:"results"`
	Total   int                `json:"total"`
	Valid   int                `json:"valid"`
	Invalid int                `json:"invalid"`
	Blocked int                `json:"blocked"` // Count of blocked URLs
}
