FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

RUN go mod tidy

# Download dependencies
RUN go mod download

RUN go get github.com/mymmrac/telego && go get github.com/mymmrac/telego/telegohandler && \
    go get gopkg.in/natefinch/lumberjack.v2

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o tg-antispam

# Use a minimal alpine image for the final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata && \
    mkdir /app

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/tg-antispam /app/

# Run the application
CMD ["/app/tg-antispam"]
