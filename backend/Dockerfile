# Multi-stage build with Debian for better package caching
FROM golang:1.23-bookworm AS builder

# Set Go proxy for faster downloads in China
ENV GOPROXY=https://goproxy.cn,direct

# Install build dependencies (faster than Alpine)
RUN apt-get update && apt-get install -y build-essential && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy dependencies first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled
ARG CGO_ENABLED=1
RUN CGO_ENABLED=1 GOOS=linux go build -a -o server ./cmd/server

# Minimal runtime image
FROM debian:bookworm-slim

# Install only runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates wget libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary and create data directory
COPY --from=builder /app/server .
RUN mkdir -p /data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget -q --spider http://localhost:8080/api/health || exit 1

CMD ["./server"]
