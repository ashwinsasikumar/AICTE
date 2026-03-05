package main

import (
	"fmt"
	"log"
	"net/http"
	"server/handlers"
	"server/middleware"
)

func main() {
	// Initialize PDF validator with 10 concurrent workers
	validator := handlers.NewPDFValidator(10)

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
	port := ":8080"
	fmt.Printf("🚀 Server started at http://localhost%s\n", port)
	fmt.Println("📝 API Endpoint: POST http://localhost:8080/api/validate")
	fmt.Println("💚 Health Check: GET http://localhost:8080/api/health")

	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
