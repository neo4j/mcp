FROM alpine:3.20 AS builder

ARG NEO4J_MCP_VERSION=1.1.0
ARG TARGETARCH

RUN apk add --no-cache curl

RUN case "${TARGETARCH}" in \
      amd64) ARCH="x86_64" ;; \
      arm64) ARCH="arm64" ;; \
      *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -fsSL "https://github.com/neo4j/mcp/releases/download/v${NEO4J_MCP_VERSION}/neo4j-mcp_Linux_${ARCH}.tar.gz" -o /tmp/neo4j-mcp.tar.gz && \
    tar -xzf /tmp/neo4j-mcp.tar.gz -C /tmp neo4j-mcp

FROM alpine:3.20

COPY --from=builder /tmp/neo4j-mcp /usr/local/bin/neo4j-mcp

ENV NEO4J_MCP_TRANSPORT=http
ENV NEO4J_MCP_HTTP_HOST=0.0.0.0
ENV NEO4J_MCP_HTTP_PORT=8080
ENV NEO4J_READ_ONLY=true
ENV NEO4J_TELEMETRY=false
ENV NEO4J_LOG_LEVEL=info

EXPOSE 8080

ENTRYPOINT ["neo4j-mcp"]
