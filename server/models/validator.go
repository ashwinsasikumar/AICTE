package models

// LinkData represents a single link with its associated college
type LinkData struct {
	College string `json:"college"`
	URL     string `json:"url"`
}

// ValidateRequest represents the incoming request with URLs to validate
type ValidateRequest struct {
	URLs []string   `json:"urls"` // Keep for backward compatibility
	Data []LinkData `json:"data"` // New field for Excel upload data
}

// ValidationResult represents the validation result for a single URL
// Status can be: "Valid", "Invalid", or "Blocked by Server"
type ValidationResult struct {
	College string `json:"college"` // Added college field
	URL     string `json:"url"`
	Status  string `json:"status"` // Simplified: "Valid", "Invalid", "Blocked by Server"
	Index   int    `json:"index"`
}

// ValidateResponse represents the complete response with all validation results
type ValidateResponse struct {
	Results []ValidationResult `json:"results"`
	Total   int                `json:"total"`
	Valid   int                `json:"valid"`
	Invalid int                `json:"invalid"`
	Blocked int                `json:"blocked"` // Count of blocked URLs
}
