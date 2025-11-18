# Builder stage
FROM golang:1.25-alpine AS builder

LABEL io.modelcontextprotocol.server.name="io.github.neo4j/mcp-neo4j"

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

# Set entrypoint
ENTRYPOINT ["/app/neo4j-mcp"]
