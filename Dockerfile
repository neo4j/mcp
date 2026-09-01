# Builder stage
FROM golang:1.26-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468 AS builder

ARG VERSION=development

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
    -ldflags "-X main.Version=${VERSION}" \
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
