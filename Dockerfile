# Alternative Dockerfile using Chromium (more reliable in containers)
# Multi-stage build for smaller production image
FROM golang:1.24-bullseye AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build for target architecture (defaults to amd64 for most servers)
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o motors-guesser cmd/server/main.go

# Production image with Chromium
FROM debian:bullseye-slim

# Install Chromium and dependencies (more stable than Chrome in containers)
RUN apt-get update && apt-get install -y \
    chromium \
    chromium-driver \
    fonts-liberation \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libatspi2.0-0 \
    libdrm2 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    xdg-utils \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user for security
RUN groupadd -r appuser && useradd -r -g appuser -G audio,video appuser \
    && mkdir -p /home/appuser/Downloads \
    && chown -R appuser:appuser /home/appuser

# Copy binary from builder stage
COPY --from=builder /app/motors-guesser /usr/local/bin/motors-guesser
COPY --from=builder /app/static /app/static

# Set permissions
RUN chmod +x /usr/local/bin/motors-guesser

# Switch to non-root user
USER appuser

# Set environment variables for Chromium
ENV CHROME_BIN=/usr/bin/chromium
ENV DISPLAY=:99
ENV CHROMIUM_FLAGS="--no-sandbox --disable-dev-shm-usage --disable-gpu --remote-debugging-port=9222"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

WORKDIR /app
EXPOSE 8080

CMD ["motors-guesser"]