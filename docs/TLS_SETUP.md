# TLS/HTTPS setup for Neo4j MCP server

This guide covers TLS/HTTPS configuration for contributors who are manually testing the Neo4j MCP server during development. It uses self-signed certificates which are not suitable for production deployments.


## Important certificate requirements


### Certificate authority

**Self-Signed Certificates**: Self-signed certificates do not work out of the box with many MCP clients (e.g., VSCode Copilot, Claude Desktop). These clients require certificates signed by a trusted Certificate Authority (CA).

**Note**: Automated tests generate certificates dynamically.

**Security**: `.pem` files are in `.gitignore` and should never be committed.


## Quickstart

### Generate a self-signed certificate

**Note**: The CN (Common Name) should match the hostname you'll use to connect.
For localhost testing, use `CN=localhost`. For a specific domain, use `CN=your-domain.com`.

```bash
# For localhost testing
openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem \
  -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost"

# For a specific domain (with SANs for proper verification)
openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem \
  -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=your-domain.com" \
  -addext "subjectAltName=DNS:your-domain.com,DNS:www.your-domain.com"
```


### Start the server with TLS

```bash
# Default port 443 when TLS is enabled
./bin/neo4j-mcp \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-transport-mode http \
  --neo4j-http-tls-enabled true \
  --neo4j-http-tls-cert-file cert.pem \
  --neo4j-http-tls-key-file key.pem

# Or specify a custom port like 8443
./bin/neo4j-mcp \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-transport-mode http \
  --neo4j-http-port 8443 \
  --neo4j-http-tls-enabled true \
  --neo4j-http-tls-cert-file cert.pem \
  --neo4j-http-tls-key-file key.pem
```


### Test the server

Use the test commands below to verify TLS setup and MCP functionality.


## Test commands


### Basic tests

```bash
# Test root path (should return 404 - server only handles /mcp)
curl -k https://127.0.0.1:8443/

# Test /mcp without authentication (should return 401)
curl -k https://127.0.0.1:8443/mcp

# Show TLS handshake details
curl -k -v https://127.0.0.1:8443/ 2>&1 | grep -E "SSL|TLS"

# Test certificate verification (should fail with self-signed cert)
curl -u neo4j:password https://127.0.0.1:8443/
```


### MCP protocol tests

```bash
# Initialize MCP session
curl -k -u neo4j:password \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {"name": "test", "version": "1.0"}
    },
    "id": 1
  }' \
  https://127.0.0.1:8443/mcp

# List available tools
curl -k -u neo4j:password \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }' \
  https://127.0.0.1:8443/mcp

# Call get-schema tool
curl -k -u neo4j:password \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "get-schema"
    },
    "id": 1
  }' \
  https://127.0.0.1:8443/mcp
```


### TLS verification

```bash
# Check TLS certificate details
openssl s_client -connect 127.0.0.1:8443 -showcerts </dev/null 2>/dev/null | openssl x509 -text -noout

# Verify TLS 1.3 support
openssl s_client -connect 127.0.0.1:8443 -tls1_3 </dev/null 2>/dev/null | grep "Protocol"

# Check cipher suites
openssl s_client -connect 127.0.0.1:8443 </dev/null 2>/dev/null | grep "Cipher"
```

## Notes

- **`-k` flag**: Skips certificate verification (needed for self-signed certificates)
- **Basic Auth**: All requests require `-u username:password`
- **Content-Type**: MCP requests need `Content-Type: application/json` header
- **Port**: Default port is 443 when TLS is enabled, 80 when TLS is disabled (configurable via `--neo4j-http-port` or `NEO4J_MCP_HTTP_PORT`)