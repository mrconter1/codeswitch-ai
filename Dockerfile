FROM golang:1.23 AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o codeswitch-ai ./cmd/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install libc compatibility layer
RUN apk add --no-cache libc6-compat

# Copy the binary from the builder
COPY --from=builder /app/codeswitch-ai .

# Ensure the binary is executable
RUN chmod +x /app/codeswitch-ai

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["/app/codeswitch-ai"]