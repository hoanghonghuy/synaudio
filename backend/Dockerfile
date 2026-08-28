# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build API and Worker binaries
RUN go build -o /app/bin/api ./cmd/api
RUN go build -o /app/bin/worker ./cmd/worker

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binaries from builder
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker

# Expose port
EXPOSE 8080

# Default command (can be overridden in docker-compose)
CMD ["/app/api"]
