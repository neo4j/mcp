# Builder stage
FROM golang:1.25-alpine@sha256:26111811bc967321e7b6f852e914d14bede324cd1accb7f81811929a6a57fea9 AS builder

LABEL io.modelcontextprotocol.server.name="io.github.neo4j/mcp"

WORKDIR /build

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

# Copy CA certificates for TLS connections to Neo4j Aura
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /build/neo4j-mcp /app/neo4j-mcp

# Run as non-root user (UID 65532 is a standard non-root user ID)
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/app/neo4j-mcp"]
