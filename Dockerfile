# Build stage
FROM golang:1.21-alpine AS builder

# Install git and SSL certificates
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o landmark-api main.go

# Final stage
FROM alpine:latest

# Add SSL certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/landmark-api .

# Copy any additional required files
COPY --from=builder /app/.env.production .env

# Set user
USER appuser

# Expose port
EXPOSE 5050

# Run the application
CMD ["./landmark-api"]