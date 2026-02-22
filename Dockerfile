# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 chaindocs && \
    adduser -D -u 1000 -G chaindocs chaindocs

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Create data directories
RUN mkdir -p /app/data /app/uploads && \
    chown -R chaindocs:chaindocs /app

# Switch to non-root user
USER chaindocs

# Expose port
EXPOSE 8080

# Volume for persistent storage
VOLUME ["/app/data", "/app/uploads"]

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/blocks/last || exit 1

# Run server
ENTRYPOINT ["/app/server"]
