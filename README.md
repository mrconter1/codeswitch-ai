# CodeSwitch AI 🔄

[![Go Version](https://img.shields.io/github/go-mod/go-version/mrconter1/codeswitch-ai)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A sophisticated code-switching service that intelligently mixes languages in text while maintaining natural readability and grammar. Currently supports English ↔ Swedish code-switching with customizable mixing ratios.

## 🌟 Features

- **Intelligent Code-Switching**: Uses frequency analysis and AI to create natural language mixing
- **Customizable Mix Ratio**: Control how much of each language appears in the output
- **Natural Grammar**: Maintains grammatical correctness across language boundaries
- **Frequency-Based Word Selection**: Uses Zipf's law and real language frequency data
- **Caching**: Built-in Redis caching for efficient repeated processing
- **Kubernetes Ready**: Full deployment configuration included

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Docker Desktop with Kubernetes enabled
- Claude API key from Anthropic

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/mrconter1/codeswitch-ai.git
cd codeswitch-ai
```

2. Create Kubernetes secret for Claude API:
```bash
kubectl create secret generic codeswitch-secrets \
  --from-literal=claude-api-key=your-api-key-here
```

3. Build and deploy:
```bash
# Build Docker image
docker build -t codeswitch-ai:latest .

# Deploy to local Kubernetes
kubectl apply -f k8s/local.yaml

# Watch the logs
kubectl logs -f deployment/codeswitch-ai
```

4. Test the service:
```bash
go run cmd/test/main.go -title="Albert_Einstein" -percent=50
```

## 🏗️ Architecture

### Components

```
├── api/              # API types and interfaces
├── cmd/
│   ├── main.go      # Main service entry point
│   └── test/        # Test client
├── internal/
│   ├── gateway/     # HTTP handlers
│   └── processor/   # Code-switching logic
├── k8s/             # Kubernetes configurations
└── pkg/
    ├── cache/       # Redis caching
    └── claude/      # Claude API client
```

## 📝 Process Flow & Pod Architecture

```typescript
// High-level request flow
async function processArticle(request) {
  // 1. Gateway Pod receives request
  // - Load balanced across gateway replicas (scales with HTTP traffic)
  // - Direct TCP connection to Redis
  gatewayPod.receive({
    title: "Albert_Einstein",
    sourceLang: "en",
    targetLang: "sv",
    switchPercent: 50
  })

  // 2. Redis Pod - Article Caching
  // - Primary pod with 2 replicas for HA
  // - Data replication across replicas
  // - Persistent volume for data storage
  const article = await redisPod.getOrFetch(request.title)
  // On cache miss, fetches from Wikipedia
  // Stores with 24h TTL

  // 3. Processor Pods - Work Queue
  // - Uses Redis pub/sub for work distribution
  // - HPA scales pods based on queue length
  // - Each pod manages own Claude API rate limits
  const paragraphs = splitIntoParagraphs(article)
  await workQueue.addBatch(paragraphs)

  // 4. Processor Pod - Each pod runs:
  class ProcessorPod {
    // Shared frequency data cached in pod memory
    // Connection pooling for Claude API calls
    freqData = loadFrequencyDictionaries()
    
    async processQueue() {
      while (true) {
        // Get next work via Redis pub/sub
        const work = await queue.getNext()
        
        // Process with Claude API
        // Rate limits managed per pod
        const result = await this.processWithClaude(work)
        
        // Mark work complete via pub/sub
        await queue.complete(work.id, result)
      }
    }
  }

  // 5. Gateway Pod - Result Assembly
  // Collects results via Redis pub/sub
  return gatewayPod.assembleAndRespond(results)
}
```

## 📝 API Reference

### Code-Switch Request
```json
POST /codeswitch
{
    "title": "Article_Title",
    "sourceLang": "en",
    "targetLang": "sv",
    "percentage": 50.0
}
```

### Response
```json
{
    "html": "<processed content>",
    "title": "Article_Title",
    "language": "sv"
}
```

## 🔧 Configuration

The service can be configured through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `CLAUDE_API_KEY` | Anthropic API key | Required |
| `REDIS_URL` | Redis connection URL | `redis://redis-service:6379` |

## 📊 Example

Input text:
```
The cat was sleeping on the table
```

With 50% Swedish code-switching:
```
The cat var sovande på det table
```

## 🛠️ Development

### Building from Source
```bash
# Get dependencies
go mod tidy

# Build
go build -o codeswitch-ai ./cmd/main.go
```

### Running Tests
```bash
go test ./...
```

### Local Development with Docker Compose
```bash
docker-compose up -d
```

## 📈 Future Improvements

- [ ] Support for more language pairs
- [ ] Advanced grammar handling
- [ ] Real-time processing mode
- [ ] Performance optimization for long texts
- [ ] Web interface for easy testing

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## ✨ Acknowledgments

- [FrequencyWords](https://github.com/hermitdave/FrequencyWords) for language frequency data
- [Anthropic](https://www.anthropic.com/) for Claude AI capabilities