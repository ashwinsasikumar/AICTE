package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"server/handlers"
	"server/middleware"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file (if it exists)
	// In production, environment variables should be set by the hosting platform
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or defaults")
	}

	// Get configuration from environment variables with defaults
	port := getEnv("PORT", "80")
	workerCount := getEnvAsInt("WORKER_COUNT", 10)

	// Initialize PDF validator with configured workers
	validator := handlers.NewPDFValidator(workerCount)

	// Create a new ServeMux for routing
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/validate", validator.ValidateLinks)

	// Health check endpoint
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","service":"PDF Link Validator"}`)
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `
			<html>
				<head><title>PDF Link Validator API</title></head>
				<body style="font-family: Arial, sans-serif; padding: 40px;">
					<h1>PDF Link Validator API</h1>
					<p>Backend server is running successfully!</p>
					<h2>Available Endpoints:</h2>
					<ul>
						<li><strong>POST /api/validate</strong> - Validate PDF links</li>
						<li><strong>GET /api/health</strong> - Health check</li>
					</ul>
					<h3>Example Request:</h3>
					<pre style="background: #f4f4f4; padding: 15px; border-radius: 5px;">
POST /api/validate
Content-Type: application/json

{
  "urls": [
    "https://example.com/document.pdf",
    "https://example.com/file.pdf"
  ]
}
					</pre>
				</body>
			</html>
		`)
	})

	// Wrap mux with CORS middleware
	handler := middleware.CORS(mux)

	// Start server
	serverPort := ":" + port
	fmt.Printf("🚀 Server started at http://localhost%s\n", serverPort)
	fmt.Printf("📝 API Endpoint: POST http://localhost%s/api/validate\n", serverPort)
	fmt.Printf("💚 Health Check: GET http://localhost%s/api/health\n", serverPort)
	fmt.Printf("👷 Worker Pool Size: %d\n", workerCount)

	if err := http.ListenAndServe(serverPort, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid value for %s, using default: %d\n", key, defaultValue)
		return defaultValue
	}
	return value
}
