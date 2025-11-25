# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o licet ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/licet .
COPY --from=builder /app/web ./web

# Copy example config (can be mounted over)
COPY config.example.yaml ./config.yaml

# Create data directory
RUN mkdir -p /app/data && chown -R app:app /app

USER app

EXPOSE 8080

CMD ["./licet"]
