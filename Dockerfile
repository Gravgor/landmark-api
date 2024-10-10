# Use the official Go image as a base
FROM golang:1.19 AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o landmark-api ./cmd/api/main.go

# Create a new image from the builder
FROM alpine:latest

# Install necessary packages (like ca-certificates)
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/landmark-api .

# Expose the port the app runs on
EXPOSE 5050

# Command to run the executable
CMD ["./landmark-api"]
