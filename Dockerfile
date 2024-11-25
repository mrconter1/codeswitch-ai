FROM golang:1.23 AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build all binaries
RUN go build -o bin/gateway ./cmd/gateway/main.go && \
    go build -o bin/processor ./cmd/processor/main.go && \
    go build -o bin/frequency-calculator ./cmd/frequency-calculator/main.go && \
    go build -o bin/result-collector ./cmd/result-collector/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install libc compatibility layer
RUN apk add --no-cache libc6-compat

# Copy binaries from builder
COPY --from=builder /app/bin/* /app/

# Ensure binaries are executable
RUN chmod +x /app/*

# Expose port 8080
EXPOSE 8080

# Command will be specified in k8s deployment
CMD ["./gateway"]