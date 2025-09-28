# Multi-stage Dockerfile for rwwwrse reverse proxy server
# Stage 1: Build stage
FROM golang:1.25-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_DATE=unknown

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy go mod files for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate Wire dependency injection code
RUN go get github.com/google/wire/cmd/wire@v0.7.0 && \
    go run github.com/google/wire/cmd/wire ./internal/di

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT_SHA} -X main.buildDate=${BUILD_DATE}" \
    -o rwwwrse \
    ./cmd/rwwwrse

# Stage 2: Runtime stage
FROM alpine:3.19

# Install runtime dependencies including Doppler CLI
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget \
    && wget -q -t3 'https://packages.doppler.com/public/cli/rsa.8004D9FF50437357.key' -O /etc/apk/keys/cli@doppler-8004D9FF50437357.rsa.pub \
    && echo 'https://packages.doppler.com/public/cli/alpine/any-version/main' | tee -a /etc/apk/repositories \
    && apk add --no-cache doppler \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1001 -S rwwwrse && \
    adduser -u 1001 -S rwwwrse -G rwwwrse

# Create directories with proper permissions
RUN mkdir -p /app/certs /app/logs && \
    chown -R rwwwrse:rwwwrse /app

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/rwwwrse /app/rwwwrse

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Change ownership and make binary executable
RUN chown rwwwrse:rwwwrse /app/rwwwrse && \
    chmod +x /app/rwwwrse

# Switch to non-root user
USER rwwwrse

# Expose ports
EXPOSE 8080 8443 9090

# Set environment variables
ENV RWWWRSE_SERVER_HOST=0.0.0.0
ENV RWWWRSE_SERVER_PORT=8080
ENV RWWWRSE_SERVER_HTTPS_PORT=8443
ENV RWWWRSE_METRICS_PORT=9090
ENV RWWWRSE_TLS_CACHE_DIR=/app/certs
ENV RWWWRSE_LOGGING_FORMAT=json
ENV RWWWRSE_LOGGING_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/rwwwrse"]

# Build metadata labels
LABEL maintainer="rwwwrse team"
LABEL org.opencontainers.image.title="rwwwrse"
LABEL org.opencontainers.image.description="Modern Go reverse proxy server with automatic TLS"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/albedosehen/rwwwrse"
LABEL org.opencontainers.image.documentation="https://github.com/albedosehen/rwwwrse/blob/main/README.md"
LABEL org.opencontainers.image.licenses="MIT"