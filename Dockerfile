# Build stage
FROM golang:1.25-alpine3.21 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Tidy dependencies and build the application
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Copy entrypoint script
COPY --from=builder /app/entrypoint.sh .
RUN chmod +x entrypoint.sh

# Expose port
EXPOSE 8080

# Set environment to production
ENV GIN_MODE=release

# Run the server via entrypoint
CMD ["./entrypoint.sh"]
