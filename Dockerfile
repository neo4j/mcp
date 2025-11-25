# Builder stage
FROM golang:1.25-alpine@sha256:d3f0cf7723f3429e3f9ed846243970b20a2de7bae6a5b66fc5914e228d831bbb AS builder

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

# Copy binary from builder
COPY --from=builder /build/neo4j-mcp /app/neo4j-mcp

# Run as non-root user (UID 65532 is a standard non-root user ID)
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/app/neo4j-mcp"]
