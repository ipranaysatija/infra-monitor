# Stage 1: build
# Use the official Go image to compile the binary
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy dependency files first - Docker caches this layer
# so 'go mod download' only re-runs when go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .

# Stage 2: minimal runtime image
# scratch has zero OS overhead - just the binary
FROM scratch

COPY --from=builder /app/exporter /exporter

# Prometheus will scrape this port
EXPOSE 8080

ENTRYPOINT ["/exporter"]
