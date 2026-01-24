# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
# No go.sum usually needed if no external deps, but good practice if I add any later
# COPY go.sum ./

COPY . .

RUN go build -o server main.go

# Final Stage
FROM alpine:latest

# Install runtime dependencies
# yt-dlp requires python3
# ffmpeg for transcoding
# curl for downloading yt-dlp
# ca-certificates for HTTPS
RUN apk add --no-cache ffmpeg python3 ca-certificates curl

# Install yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod +x /usr/local/bin/yt-dlp

WORKDIR /app

COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

CMD ["./server"]
