package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"server/models"
	"strings"
	"sync"
	"time"
)

// PDFValidator handles PDF link validation with intelligent fallback logic
type PDFValidator struct {
	client      *http.Client
	workerCount int
}

// NewPDFValidator creates a new PDF validator with configured HTTP client
func NewPDFValidator(workerCount int) *PDFValidator {
	// Create custom transport to disable HTTP/2 and configure connection settings
	transport := &http.Transport{
		// Disable HTTP/2 by setting TLSNextProto to empty map - forces HTTP/1.1
		TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),

		// TLS configuration for secure connections
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
		},

		// Connection pooling and keep-alive settings
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,

		// Timeouts for various stages
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Disable compression to avoid issues with partial content
		DisableCompression: false,
	}

	return &PDFValidator{
		client: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second, // Overall client timeout
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Prevent infinite redirect loops - max 5 redirects
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				return nil
			},
		},
		workerCount: workerCount,
	}
}

// ValidateLinks handles the POST /api/validate endpoint
func (v *PDFValidator) ValidateLinks(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	var results []models.ValidationResult
	var dataToValidate []models.LinkData

	// Check if using new format (with college data) or old format (just URLs)
	if len(req.Data) > 0 {
		// New format with college information
		dataToValidate = req.Data
		if len(dataToValidate) > 500 {
			http.Error(w, "Too many URLs. Maximum 500 URLs per request", http.StatusBadRequest)
			return
		}
	} else if len(req.URLs) > 0 {
		// Old format for backward compatibility
		if len(req.URLs) > 100 {
			http.Error(w, "Too many URLs. Maximum 100 URLs per request", http.StatusBadRequest)
			return
		}
		// Convert to LinkData format with empty college
		for _, url := range req.URLs {
			dataToValidate = append(dataToValidate, models.LinkData{
				College: "",
				URL:     url,
			})
		}
	} else {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	// Validate URLs concurrently using worker pool
	results = v.validateConcurrentlyWithCollege(dataToValidate)

	// Count valid, invalid, and blocked
	validCount := 0
	invalidCount := 0
	blockedCount := 0
	for _, result := range results {
		switch result.Status {
		case "Valid":
			validCount++
		case "Invalid":
			invalidCount++
		case "Blocked by Server":
			blockedCount++
		}
	}

	// Build response
	response := models.ValidateResponse{
		Results: results,
		Total:   len(results),
		Valid:   validCount,
		Invalid: invalidCount,
		Blocked: blockedCount,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// validateConcurrently validates multiple URLs using a worker pool pattern
func (v *PDFValidator) validateConcurrently(urls []string) []models.ValidationResult {
	results := make([]models.ValidationResult, len(urls))
	jobs := make(chan int, len(urls))
	var wg sync.WaitGroup

	// Start worker pool
	for w := 0; w < v.workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				results[index] = v.validateSingleURL(urls[index], index, "")
			}
		}()
	}

	// Send jobs to workers
	for i := range urls {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()

	return results
}

// validateConcurrentlyWithCollege validates multiple URLs with college info using a worker pool pattern
func (v *PDFValidator) validateConcurrentlyWithCollege(data []models.LinkData) []models.ValidationResult {
	results := make([]models.ValidationResult, len(data))
	jobs := make(chan int, len(data))
	var wg sync.WaitGroup

	// Start worker pool
	for w := 0; w < v.workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				results[index] = v.validateSingleURL(data[index].URL, index, data[index].College)
			}
		}()
	}

	// Send jobs to workers
	for i := range data {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()

	return results
}

// internalValidationResult holds detailed validation information for internal use
type internalValidationResult struct {
	statusCode  int
	contentType string
	isValid     bool
	reason      string
}

// mapToSimplifiedStatus converts detailed internal validation to simplified user-facing status
// Returns one of: "Valid", "Invalid", "Blocked by Server"
func mapToSimplifiedStatus(internal internalValidationResult, urlStr string) models.ValidationResult {
	result := models.ValidationResult{
		URL: urlStr,
	}

	// Log detailed information for debugging (not exposed to frontend)
	log.Printf("[VALIDATION] URL: %s | StatusCode: %d | ContentType: %s | IsValid: %v | Reason: %s",
		urlStr, internal.statusCode, internal.contentType, internal.isValid, internal.reason)

	// Check if this is a "Blocked by Server" case
	if isBlockedByServer(internal) {
		result.Status = "Blocked by Server"
		return result
	}

	// Check if valid PDF
	if internal.isValid {
		result.Status = "Valid"
		return result
	}

	// Everything else is invalid (wrong content type, 404, 500, etc.)
	result.Status = "Invalid"
	return result
}

// isBlockedByServer determines if the failure is due to network-level blocking/errors
// Returns true for: HTTP 403, timeouts, DNS failures, TLS errors, connection issues
// These are cases where the server/network prevents access, not content validation failures
func isBlockedByServer(internal internalValidationResult) bool {
	reason := strings.ToLower(internal.reason)

	// HTTP 403 Forbidden - explicit server blocking
	if internal.statusCode == 403 {
		return true
	}

	// Network-level failure patterns (not content validation failures)
	blockingPatterns := []string{
		// Timeout and deadline errors
		"timeout",
		"deadline exceeded",
		"context deadline",

		// DNS resolution failures
		"dns resolution",
		"no such host",

		// Connection failures
		"connection refused",
		"connection reset",
		"connection timed out",

		// TLS/SSL errors
		"ssl/tls error",
		"tls handshake",
		"certificate",
		"ssl error",

		// Protocol errors
		"http/2 stream error",
		"protocol error",
		"stream error",

		// Access control / CDN blocking
		"blocked",
		"cloudflare",
		"akamai",
		"access denied",
		"forbidden",
	}

	for _, pattern := range blockingPatterns {
		if strings.Contains(reason, pattern) {
			return true
		}
	}

	return false
}

// validateSingleURL validates a single URL and returns the result
// Uses intelligent fallback: HEAD request first, then partial GET if HEAD fails
func (v *PDFValidator) validateSingleURL(urlStr string, index int, college string) models.ValidationResult {
	// Trim whitespace from URL
	urlStr = strings.TrimSpace(urlStr)

	// Skip empty URLs
	if urlStr == "" {
		internal := internalValidationResult{
			statusCode:  0,
			contentType: "",
			isValid:     false,
			reason:      "Empty URL",
		}
		result := mapToSimplifiedStatus(internal, urlStr)
		result.Index = index + 1
		result.College = college
		return result
	}

	// Validate URL format using url.ParseRequestURI
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		internal := internalValidationResult{
			statusCode:  0,
			contentType: "",
			isValid:     false,
			reason:      fmt.Sprintf("Invalid URL format: %v", err),
		}
		result := mapToSimplifiedStatus(internal, urlStr)
		result.Index = index + 1
		result.College = college
		return result
	}

	// Ensure URL has http:// or https:// scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		internal := internalValidationResult{
			statusCode:  0,
			contentType: "",
			isValid:     false,
			reason:      "URL must use http:// or https:// scheme",
		}
		result := mapToSimplifiedStatus(internal, urlStr)
		result.Index = index + 1
		result.College = college
		return result
	}

	// Try HEAD request first (fast, but may fail on some servers)
	headSuccess, headInternal := v.tryHEADRequest(urlStr)

	if headSuccess {
		// HEAD request succeeded and gave us a definitive answer
		result := mapToSimplifiedStatus(headInternal, urlStr)
		result.Index = index + 1
		result.College = college
		return result
	}

	// HEAD failed or was inconclusive - fall back to partial GET request
	getInternal := v.tryPartialGETRequest(urlStr, headInternal)
	result := mapToSimplifiedStatus(getInternal, urlStr)
	result.Index = index + 1
	result.College = college
	return result
}

// setBrowserHeaders sets comprehensive browser-like headers to bypass CDN protection
func (v *PDFValidator) setBrowserHeaders(req *http.Request, referer string) {
	// Modern Chrome User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Accept headers to mimic browser behavior
	req.Header.Set("Accept", "application/pdf,application/octet-stream,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	// Connection settings for keep-alive
	req.Header.Set("Connection", "keep-alive")

	// Security and origin headers
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Cache control
	req.Header.Set("Cache-Control", "max-age=0")

	// Optional referer if provided
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

// tryHEADRequest attempts a HEAD request to validate the URL
// Returns (success, internalResult) where success=true means we got a definitive answer
// Priority order: (1) Network error → Blocked, (2) Status code check, (3) Content validation
func (v *PDFValidator) tryHEADRequest(url string) (bool, internalValidationResult) {
	result := internalValidationResult{}

	// Create context with 10-second timeout for HEAD request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	headReq, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		// Invalid URL format - don't fall back, this is a real error
		result.isValid = false
		result.reason = fmt.Sprintf("Invalid URL format: %v", err)
		return true, result // Definitive answer
	}

	// Set comprehensive browser-like headers to bypass CDN protection
	v.setBrowserHeaders(headReq, "")

	resp, err := v.client.Do(headReq)

	// STAGE 1: Check for network-level errors first (all return Blocked)
	if err != nil {
		errStr := err.Error()

		// Explicit check: Context deadline exceeded (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			result.isValid = false
			result.statusCode = 0
			result.reason = "Timeout - Context deadline exceeded"
			return true, result // Definitive: Blocked
		}

		// Explicit check: Client timeout
		if strings.Contains(errStr, "Client.Timeout exceeded") {
			result.isValid = false
			result.statusCode = 0
			result.reason = "Timeout - Client timeout exceeded"
			return true, result // Definitive: Blocked
		}

		// Explicit check: DNS resolution failure
		if strings.Contains(errStr, "no such host") {
			result.isValid = false
			result.statusCode = 0
			result.reason = "DNS resolution failed"
			return true, result // Definitive: Blocked
		}

		// Explicit check: TLS handshake failure
		if strings.Contains(errStr, "TLS") || strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate") {
			result.isValid = false
			result.statusCode = 0
			result.reason = "TLS handshake failure"
			return true, result // Definitive: Blocked
		}

		// Explicit check: Connection refused
		if strings.Contains(errStr, "connection refused") {
			result.isValid = false
			result.statusCode = 0
			result.reason = "Connection refused"
			return true, result // Definitive: Blocked
		}

		// Explicit check: Connection reset
		if strings.Contains(errStr, "connection reset") {
			result.isValid = false
			result.statusCode = 0
			result.reason = "Connection reset"
			return true, result // Definitive: Blocked
		}

		// Explicit check: Connection timeout
		if strings.Contains(errStr, "connection timed out") || strings.Contains(errStr, "i/o timeout") {
			result.isValid = false
			result.statusCode = 0
			result.reason = "Connection timeout"
			return true, result // Definitive: Blocked
		}

		// Protocol errors that may indicate method not supported - try GET
		if strings.Contains(errStr, "INTERNAL_ERROR") || strings.Contains(errStr, "stream") {
			result.reason = "HTTP/2 stream error, trying partial GET"
			return false, result // Try GET fallback
		}

		if strings.Contains(errStr, "protocol") {
			result.reason = "Protocol error, trying partial GET"
			return false, result // Try GET fallback
		}

		// Other network errors - try GET as fallback
		result.reason = fmt.Sprintf("HEAD failed (%v), trying partial GET", err)
		return false, result
	}
	defer resp.Body.Close()

	result.statusCode = resp.StatusCode
	result.contentType = resp.Header.Get("Content-Type")

	// STAGE 2: Explicit status code checks (no range-based grouping)

	// Explicit check: 403 Forbidden → Blocked (must be first, before other status codes)
	if resp.StatusCode == 403 {
		// Try GET anyway since some servers block HEAD but allow GET
		result.statusCode = 403
		result.reason = "HTTP 403 on HEAD, trying partial GET"
		return false, result // Try GET, but preserve 403
	}

	// Explicit check: Method Not Allowed - server doesn't support HEAD
	if resp.StatusCode == 405 {
		result.reason = "HEAD not allowed, trying partial GET"
		return false, result
	}

	// Explicit check: Not Implemented
	if resp.StatusCode == 501 {
		result.reason = "HEAD not implemented, trying partial GET"
		return false, result
	}

	// Explicit check: 404 Not Found → Invalid
	if resp.StatusCode == 404 {
		result.isValid = false
		result.reason = "Invalid - HTTP 404 Not Found"
		return true, result // Definitive: Invalid
	}

	// Explicit check: 500 Internal Server Error → Invalid
	if resp.StatusCode == 500 {
		result.isValid = false
		result.reason = "Invalid - HTTP 500 Internal Server Error"
		return true, result // Definitive: Invalid
	}

	// Explicit check: 502 Bad Gateway → Invalid
	if resp.StatusCode == 502 {
		result.isValid = false
		result.reason = "Invalid - HTTP 502 Bad Gateway"
		return true, result // Definitive: Invalid
	}

	// Explicit check: 503 Service Unavailable → Invalid
	if resp.StatusCode == 503 {
		result.isValid = false
		result.reason = "Invalid - HTTP 503 Service Unavailable"
		return true, result // Definitive: Invalid
	}

	// Explicit check: 504 Gateway Timeout → Invalid
	if resp.StatusCode == 504 {
		result.isValid = false
		result.reason = "Invalid - HTTP 504 Gateway Timeout"
		return true, result // Definitive: Invalid
	}

	// Other non-200 status codes → Invalid
	if resp.StatusCode != 200 {
		result.isValid = false
		result.reason = fmt.Sprintf("Invalid - HTTP %d", resp.StatusCode)
		return true, result // Definitive: Invalid
	}

	// STAGE 3: Status is 200, now validate content
	// This stage only runs for successful HTTP 200 responses

	// Parse content type
	contentType := strings.ToLower(result.contentType)
	contentType = strings.Split(contentType, ";")[0] // Remove charset and other parameters
	contentType = strings.TrimSpace(contentType)

	// Explicit check: Content-Type contains "application/pdf" → Valid
	if strings.Contains(contentType, "application/pdf") {
		result.isValid = true
		result.reason = "Valid PDF via HEAD"
		return true, result // Definitive: Valid
	}

	// Explicit check: HTML page with 200 response → Invalid (not Blocked)
	if strings.Contains(contentType, "text/html") {
		result.isValid = false
		result.reason = "Invalid - HTML page"
		return true, result // Definitive: Invalid
	}

	// Explicit check: Plain text with 200 response → Invalid
	if strings.Contains(contentType, "text/plain") {
		result.isValid = false
		result.reason = "Invalid - Plain text file"
		return true, result // Definitive: Invalid
	}

	// Explicit check: Video file with 200 response → Invalid
	if strings.Contains(contentType, "video/") {
		result.isValid = false
		result.reason = "Invalid - Video file"
		return true, result // Definitive: Invalid
	}

	// Explicit check: Audio file with 200 response → Invalid
	if strings.Contains(contentType, "audio/") {
		result.isValid = false
		result.reason = "Invalid - Audio file"
		return true, result // Definitive: Invalid
	}

	// Explicit check: Image file with 200 response → Invalid
	if strings.Contains(contentType, "image/") {
		result.isValid = false
		result.reason = "Invalid - Image file"
		return true, result // Definitive: Invalid
	}

	// Content-Type is missing, empty, or application/octet-stream
	// Need to check actual file content with GET request
	if contentType == "" || contentType == "application/octet-stream" {
		result.reason = "Ambiguous Content-Type, checking file signature with GET"
		return false, result // Not definitive, need to check signature
	}

	// Unknown content type - fall back to GET to be sure
	result.reason = fmt.Sprintf("Suspicious Content-Type (%s), verifying with GET", contentType)
	return false, result
}

// tryPartialGETRequest performs a partial GET request to check PDF signature
// This is the fallback when HEAD request fails or is inconclusive
// Priority order: (1) Network error → Blocked, (2) Status code check, (3) Content validation
func (v *PDFValidator) tryPartialGETRequest(url string, previousResult internalValidationResult) internalValidationResult {
	result := internalValidationResult{
		statusCode:  previousResult.statusCode,
		contentType: previousResult.contentType,
	}

	// Create context with 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.isValid = false
		result.reason = fmt.Sprintf("Invalid URL: %v", err)
		return result
	}

	// Request only first 2KB to check headers and signature
	req.Header.Set("Range", "bytes=0-2048")

	// Set comprehensive browser-like headers to bypass CDN protection
	v.setBrowserHeaders(req, url)

	resp, err := v.client.Do(req)

	// STAGE 1: Explicit network error checks (all return Blocked immediately)
	if err != nil {
		result.isValid = false
		result.statusCode = 0
		errStr := err.Error()

		// Explicit check: Context deadline exceeded (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			result.reason = "Timeout - Context deadline exceeded"
			return result // Blocked
		}

		// Explicit check: Client timeout
		if strings.Contains(errStr, "Client.Timeout exceeded") {
			result.reason = "Timeout - Client timeout exceeded"
			return result // Blocked
		}

		// Explicit check: DNS resolution failure
		if strings.Contains(errStr, "no such host") {
			result.reason = "DNS resolution failed"
			return result // Blocked
		}

		// Explicit check: TLS handshake failure
		if strings.Contains(errStr, "TLS") || strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate") {
			result.reason = "TLS handshake failure"
			return result // Blocked
		}

		// Explicit check: Connection refused
		if strings.Contains(errStr, "connection refused") {
			result.reason = "Connection refused"
			return result // Blocked
		}

		// Explicit check: Connection reset
		if strings.Contains(errStr, "connection reset") {
			result.reason = "Connection reset"
			return result // Blocked
		}

		// Explicit check: Connection timeout
		if strings.Contains(errStr, "connection timed out") || strings.Contains(errStr, "i/o timeout") {
			result.reason = "Connection timeout"
			return result // Blocked
		}

		// Explicit check: HTTP/2 stream error
		if strings.Contains(errStr, "INTERNAL_ERROR") || strings.Contains(errStr, "stream") {
			result.reason = "HTTP/2 stream error"
			return result // Blocked
		}

		// Explicit check: Protocol error
		if strings.Contains(errStr, "protocol") {
			result.reason = "Protocol error"
			return result // Blocked
		}

		// Explicit check: General timeout
		if strings.Contains(errStr, "timeout") {
			result.reason = "Timeout"
			return result // Blocked
		}

		// Generic network error (any other transport error)
		result.reason = fmt.Sprintf("Network error: %v", err)
		return result // Blocked
	}
	defer resp.Body.Close()

	// Update status code and content type from GET response
	result.statusCode = resp.StatusCode
	result.contentType = resp.Header.Get("Content-Type")

	// STAGE 2: Explicit status code checks (no range-based grouping)
	// Each status code is checked individually with immediate return

	// Explicit check: 403 Forbidden → Blocked (MUST be checked first, before any other status)
	if resp.StatusCode == 403 {
		result.isValid = false
		result.statusCode = 403
		result.reason = "HTTP 403 Forbidden - Access denied"
		return result // Blocked
	}

	// Explicit check: 404 Not Found → Invalid
	if resp.StatusCode == 404 {
		result.isValid = false
		result.reason = "Invalid - HTTP 404 Not Found"
		return result // Invalid
	}

	// Explicit check: 500 Internal Server Error → Invalid
	if resp.StatusCode == 500 {
		result.isValid = false
		result.reason = "Invalid - HTTP 500 Internal Server Error"
		return result // Invalid
	}

	// Explicit check: 502 Bad Gateway → Invalid
	if resp.StatusCode == 502 {
		result.isValid = false
		result.reason = "Invalid - HTTP 502 Bad Gateway"
		return result // Invalid
	}

	// Explicit check: 503 Service Unavailable → Invalid
	if resp.StatusCode == 503 {
		result.isValid = false
		result.reason = "Invalid - HTTP 503 Service Unavailable"
		return result // Invalid
	}

	// Explicit check: 504 Gateway Timeout → Invalid
	if resp.StatusCode == 504 {
		result.isValid = false
		result.reason = "Invalid - HTTP 504 Gateway Timeout"
		return result // Invalid
	}

	// Explicit check: Only 200 and 206 proceed to content validation
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		result.isValid = false
		result.reason = fmt.Sprintf("Invalid - HTTP %d", resp.StatusCode)
		return result // Invalid
	}

	// STAGE 3: Status is 200 or 206, perform content validation
	// This stage distinguishes Valid from Invalid for successful HTTP responses
	// All results here are either Valid or Invalid (never Blocked)

	// Parse content type from GET response
	contentType := strings.ToLower(result.contentType)
	contentType = strings.Split(contentType, ";")[0]
	contentType = strings.TrimSpace(contentType)

	// Explicit check: Content-Type is "application/pdf" → verify with signature
	if strings.Contains(contentType, "application/pdf") {
		// Read first bytes to confirm PDF signature
		buffer := make([]byte, 32)
		n, err := io.ReadAtLeast(resp.Body, buffer, 5)
		if err != nil && err != io.ErrUnexpectedEOF {
			result.isValid = false
			result.reason = "Failed to read response body"
			return result // Invalid (read error on 200 response)
		}

		// Check for PDF signature: %PDF-
		pdfSignature := []byte("%PDF-")
		hasPDFSignature := n >= len(pdfSignature) && bytes.Equal(buffer[:len(pdfSignature)], pdfSignature)

		// Explicit branch: Signature found → Valid
		if hasPDFSignature {
			result.isValid = true
			result.reason = "Valid PDF - Content-Type and signature confirmed"
			return result // Valid
		}

		// Explicit branch: Signature not found → Invalid
		result.isValid = false
		result.reason = "Invalid - Content-Type claims PDF but signature missing"
		return result // Invalid
	}

	// Explicit check: Content-Type is "application/octet-stream" or empty → check signature
	if strings.Contains(contentType, "application/octet-stream") || contentType == "" {
		// Read first bytes to check for PDF signature
		buffer := make([]byte, 32)
		n, err := io.ReadAtLeast(resp.Body, buffer, 5)
		if err != nil && err != io.ErrUnexpectedEOF {
			result.isValid = false
			result.reason = "Failed to read response body"
			return result // Invalid (read error on 200 response)
		}

		// Check for PDF signature: %PDF-
		pdfSignature := []byte("%PDF-")
		hasPDFSignature := n >= len(pdfSignature) && bytes.Equal(buffer[:len(pdfSignature)], pdfSignature)

		// Explicit branch: Signature found → Valid
		if hasPDFSignature {
			result.isValid = true
			result.reason = "Valid PDF - Signature confirmed"
			return result // Valid
		}

		// Explicit branch: Signature not found → Invalid
		result.isValid = false
		result.reason = "Invalid - Not a PDF"
		return result // Invalid
	}

	// Explicit check: HTML page with 200 response → Invalid (not Blocked)
	if strings.Contains(contentType, "text/html") {
		result.isValid = false
		result.reason = "Invalid - HTML page"
		return result // Invalid
	}

	// Explicit check: Text file with 200 response → Invalid
	if strings.Contains(contentType, "text/") {
		result.isValid = false
		result.reason = "Invalid - Text file"
		return result // Invalid
	}

	// Explicit check: Video file with 200 response → Invalid
	if strings.Contains(contentType, "video/") {
		result.isValid = false
		result.reason = "Invalid - Video file"
		return result // Invalid
	}

	// Explicit check: Audio file with 200 response → Invalid
	if strings.Contains(contentType, "audio/") {
		result.isValid = false
		result.reason = "Invalid - Audio file"
		return result // Invalid
	}

	// Explicit check: Image file with 200 response → Invalid
	if strings.Contains(contentType, "image/") {
		result.isValid = false
		result.reason = "Invalid - Image file"
		return result // Invalid
	}

	// Final explicit check: Unknown content type with 200 response → Invalid
	// (Successful response with non-PDF content is always Invalid, never Blocked)
	result.isValid = false
	result.reason = "Invalid - Not a PDF"
	return result // Invalid
}
