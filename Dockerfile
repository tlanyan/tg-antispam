FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o tg-antispam ./cmd/tg-antispam

# Use a minimal alpine image for the final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata && \
    mkdir -p /app/logs

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/tg-antispam /app/

# Copy configs
COPY --from=builder /app/configs /app/configs

# Set executable permissions
RUN chmod +x /app/tg-antispam

# Run the application
CMD ["/app/tg-antispam"]
