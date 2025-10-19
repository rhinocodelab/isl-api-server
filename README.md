# ISL API Server

A high-performance, concurrent Go API server for text translation using Google Cloud Translation API with support for multiple Indian languages.

## 🚀 Features

- **Concurrent Translation**: Goroutines + WaitGroup for 3x faster multi-language translations
- **Partial Success Handling**: Returns available translations even if some fail
- **Robust Error Handling**: Individual language error isolation
- **Structured Logging**: Comprehensive logging to file with context
- **Health Monitoring**: Built-in health check endpoint
- **CORS Support**: Cross-origin resource sharing enabled
- **Graceful Shutdown**: Proper resource cleanup on termination

## 🏗️ Architecture

### Service-Oriented Design
```
├── cmd/server/           # Application entry point
├── config/              # Configuration management
├── internal/
│   ├── models/          # Data models and DTOs
│   ├── router/          # HTTP routing and middleware
│   ├── service/         # Business logic services
│   └── util/            # Utility functions (logging)
└── log/                 # Application logs
```

### Supported Languages
- **English** (en)
- **Hindi** (hi) - हिन्दी
- **Marathi** (mr) - मराठी
- **Gujarati** (gu) - ગુજરાતી

## 📋 API Endpoints

### Health Check
```http
GET /api/v1/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-19T10:30:00Z",
  "version": "1.0.0"
}
```

### Text Translation
```http
POST /api/v1/text-translate
Content-Type: application/json

{
  "text": "Hello World",
  "source_language": "en"
}
```

**Response:**
```json
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": {
    "hi": "नमस्ते दुनिया",
    "mr": "हॅलो वर्ल्ड",
    "gu": "હેલો વર્લ્ડ"
  }
}
```

### API Information
```http
GET /
```

**Response:**
```json
{
  "message": "Welcome to ISL API Server",
  "version": "1.0.0",
  "endpoints": {
    "health": "/api/v1/health",
    "text-translate": "/api/v1/text-translate"
  }
}
```

## ⚡ Performance Benefits

### Concurrent Translation
- **3x Faster**: Parallel execution vs sequential
- **Partial Success**: Returns available translations even with failures
- **Error Isolation**: Individual language failures don't affect others
- **Timeout Protection**: 30-second timeout per language

### Performance Comparison

| Scenario | Sequential | Concurrent | Improvement |
|----------|------------|------------|-------------|
| **3 Languages** | ~3 seconds | ~1 second | **3x faster** |
| **All Success** | 3 seconds | 1 second | **3x faster** |
| **1 Failure** | Fails completely | 2 successes | **Partial success** |
| **All Failures** | Fails immediately | Fails with details | **Better error info** |

## 🛠️ Installation & Setup

### Prerequisites
- Go 1.19 or higher
- Google Cloud Project with Translation API enabled
- GCP Service Account with Translation API permissions

### 1. Clone Repository
```bash
git clone <repository-url>
cd isl-api-server
```

### 2. Install Dependencies
```bash
go mod tidy
```

### 3. Configure Environment
Copy the configuration template and create a `.env` file:
```bash
cp config.env.template .env
```

Update the `.env` file with your values:
```bash
# GCP Configuration
GCP_PROJECT_ID=your-project-id
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

# Server Configuration
PORT=5001
ENVIRONMENT=development
LOG_PATH=./log/isl-api-server.log

# Language Dictionary Configuration
LANGUAGE_DICTIONARY_PATH=./config/language_dictionary

# Timeout Configuration
READ_TIMEOUT=10
WRITE_TIMEOUT=10
IDLE_TIMEOUT=120
```

### 4. Setup GCP Credentials
1. Create a service account in your GCP project
2. Assign "Cloud Translation API User" role
3. Download the JSON key file
4. Place it in `config/credentials/` directory
5. Update `GOOGLE_APPLICATION_CREDENTIALS` path in `.env`

### 5. Run the Server
```bash
# Development
go run cmd/server/main.go

# Production
go build -o isl-api-server cmd/server/main.go
./isl-api-server
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GCP_PROJECT_ID` | Google Cloud Project ID | Required |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to GCP credentials JSON | Required |
| `PORT` | Server port | 5001 |
| `ENVIRONMENT` | Environment (development/production) | development |
| `LOG_PATH` | Log file path | ./log/isl-api-server.log |
| `LANGUAGE_DICTIONARY_PATH` | Path to language dictionary files | ./config/language_dictionary |
| `READ_TIMEOUT` | HTTP read timeout (seconds) | 10 |
| `WRITE_TIMEOUT` | HTTP write timeout (seconds) | 10 |
| `IDLE_TIMEOUT` | HTTP idle timeout (seconds) | 120 |

### GCP Setup
1. Enable Cloud Translation API in your GCP project
2. Create a service account
3. Assign "Cloud Translation API User" role
4. Download and configure credentials

## 📊 Monitoring & Logging

### Structured Logging
All operations are logged with structured data:
```
INFO: Starting concurrent multi-language translation source=en targets=[hi,mr,gu] text_length=11
INFO: Translation completed target=hi translated_length=15
ERROR: Translation failed for language target=mr error=<error>
INFO: Concurrent multi-language translation completed translations_count=2 errors_count=1
```

### Log Levels
- **INFO**: Normal operations, successful translations
- **ERROR**: Translation failures, service errors
- **FATAL**: Critical errors, server shutdown

### Health Monitoring
- Built-in health check endpoint
- Service status monitoring
- Performance metrics tracking

## 🧪 Testing

### Manual Testing
```bash
# Health check
curl http://localhost:5001/api/v1/health

# Text translation
curl -X POST http://localhost:5001/api/v1/text-translate \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello World", "source_language": "en"}'
```

### Load Testing
```bash
# Install hey (load testing tool)
go install github.com/rakyll/hey@latest

# Run load test
hey -n 100 -c 10 -m POST -H "Content-Type: application/json" \
  -d '{"text": "Hello World", "source_language": "en"}' \
  http://localhost:5001/api/v1/text-translate
```

## 🚀 Deployment

### Docker Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o isl-api-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/isl-api-server .
COPY --from=builder /app/.env .
EXPOSE 5001
CMD ["./isl-api-server"]
```

### Production Considerations
- Set `ENVIRONMENT=production` for optimized logging
- Configure proper log rotation
- Set up monitoring and alerting
- Use reverse proxy (nginx) for production
- Configure SSL/TLS certificates

## 🔍 Troubleshooting

### Common Issues

#### GCP Authentication Errors
```
ERROR: Translation failed error=rpc error: code = Unauthenticated
```
**Solution**: Verify GCP credentials and project ID configuration

#### Service Unavailable
```
ERROR: Translation service unavailable
```
**Solution**: Check GCP_PROJECT_ID environment variable

#### Translation Failures
```
ERROR: Translation failed for language target=hi error=<error>
```
**Solution**: Check GCP Translation API quotas and billing

### Debug Mode
Enable debug logging by setting environment to development:
```bash
export ENVIRONMENT=development
```

## 📚 Documentation

- [Text Translation Service](./internal/service/doc/text_translation_service.md)
- [Language Service](./internal/service/doc/language_service.md)
- [Text Translation Models](./internal/models/doc/text_translation_models.md)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the logs in `log/isl-api-server.log`

## 🎯 Roadmap

- [ ] Add more Indian languages support
- [ ] Implement caching for translations
- [ ] Add rate limiting
- [ ] Implement authentication/authorization
- [ ] Add metrics and monitoring dashboard
- [ ] Support for batch translation requests

---

**Built with ❤️ using Go, Gin, and Google Cloud Translation API**
