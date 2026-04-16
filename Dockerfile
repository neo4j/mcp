# Builder stage
FROM golang:1.26-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

LABEL io.modelcontextprotocol.server.name="io.github.neo4j/mcp"

WORKDIR /build

# Install CA certificates
RUN apk add --no-cache ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -C cmd/neo4j-mcp -a -installsuffix cgo \
    -o ../../neo4j-mcp

# Runtime stage
FROM scratch

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/neo4j-mcp /app/neo4j-mcp

# Copy CA certificates for HTTPS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Run as non-root user (UID 65532 is a standard non-root user ID)
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/app/neo4j-mcp"]
