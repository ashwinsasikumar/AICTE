# Go Backend - PDF Link Validator API

RESTful API built with Go for validating PDF links concurrently.

## Features

- ✅ Concurrent validation using worker pools
- ✅ HTTP HEAD requests for efficiency
- ✅ PDF signature detection
- ✅ Timeout handling (5 seconds)
- ✅ Redirect management (max 10 redirects)
- ✅ CORS support
- ✅ Comprehensive error handling

## Project Structure

```
server/
├── handlers/
│   └── validator.go      # PDF validation logic
├── middleware/
│   └── cors.go           # CORS middleware
├── models/
│   └── validator.go      # Data models
├── main.go               # Entry point & routing
├── go.mod                # Dependencies
├── Dockerfile            # Docker configuration
└── .gitignore            # Git ignore rules
```

## Running Locally

```bash
# Install dependencies
go mod tidy

# Run the server
go run main.go

# Or build and run
go build -o pdf-validator
./pdf-validator
```

Server will start on `http://localhost:8080`

## API Endpoints

### POST /api/validate
Validates PDF links

**Request:**
```json
{
  "urls": ["https://example.com/file.pdf"]
}
```

**Response:**
```json
{
  "results": [{
    "url": "https://example.com/file.pdf",
    "statusCode": 200,
    "contentType": "application/pdf",
    "isValid": true,
    "reason": "Valid PDF",
    "index": 1
  }],
  "total": 1,
  "valid": 1,
  "invalid": 0
}
```

### GET /api/health
Health check endpoint

## Configuration

- **Port**: 8080 (change in main.go)
- **Workers**: 10 concurrent workers (change in main.go)
- **Timeout**: 5 seconds per request (change in handlers/validator.go)
- **Max URLs**: 100 per request (change in handlers/validator.go)

## Docker

```bash
# Build image
docker build -t pdf-validator-backend .

# Run container
docker run -p 8080:8080 pdf-validator-backend
```

## Dependencies

- Standard library only (no external dependencies)
- Go 1.24+

## Testing

```bash
# Test health endpoint
curl http://localhost:8080/api/health

# Test validation
curl -X POST http://localhost:8080/api/validate \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"]}'
```
