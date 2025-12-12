# TLS Testing for Neo4j MCP Server

**Note**: Automated tests generate certificates dynamically. For manual/integration testing, follow the steps below.

**Security**: `.pem` files are in `.gitignore` and should never be committed.

## Quick Start

### 1. Generate Self-Signed Certificate (For Manual Testing)

**Note**: The CN (Common Name) should match the hostname you'll use to connect. For localhost testing, use `CN=localhost`. For a specific domain, use `CN=your-domain.com`.

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

### 2. Start the Server with TLS

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

Or using environment variables:

```bash
export NEO4J_URI="bolt://localhost:7687"
# Note: In HTTP mode, NEO4J_USERNAME and NEO4J_PASSWORD are not used
# Credentials come from per-request Basic Auth headers
export NEO4J_MCP_TRANSPORT="http"
export NEO4J_MCP_HTTP_TLS_ENABLED="true"
export NEO4J_MCP_HTTP_TLS_CERT_FILE="cert.pem"
export NEO4J_MCP_HTTP_TLS_KEY_FILE="key.pem"
# NEO4J_MCP_HTTP_PORT defaults to 443 when TLS is enabled

./bin/neo4j-mcp
```

### 3. Test the Server

Use the test commands below to verify TLS setup and MCP functionality.

## Test Commands

### Basic Tests

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

### MCP Protocol Tests

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

### TLS Verification

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
- **Port**: Default port is 443 when TLS is enabled, 8080 when TLS is disabled (configurable via `--neo4j-http-port` or `NEO4J_MCP_HTTP_PORT`)

## Production Use

For production, use a proper certificate from a Certificate Authority (e.g., Let's Encrypt).

**Important**: The certificate's Common Name (CN) and Subject Alternative Names (SANs) must match the domain name clients will use to connect. Let's Encrypt certificates automatically include the correct domain names.

```bash
# With Let's Encrypt certificate (certificates include proper domain names)
# Note: In HTTP mode, username/password are not needed here - credentials come from per-request Basic Auth
./bin/neo4j-mcp \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-transport-mode http \
  --neo4j-http-host 0.0.0.0 \
  --neo4j-http-port 443 \
  --neo4j-http-tls-enabled true \
  --neo4j-http-tls-cert-file /etc/letsencrypt/live/your-domain.com/fullchain.pem \
  --neo4j-http-tls-key-file /etc/letsencrypt/live/your-domain.com/privkey.pem
```

Then clients can connect using the domain name without `-k` flag:

```bash
# Connect to /mcp endpoint (the only valid path)
curl -u neo4j:password https://your-domain.com/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}'

# Other paths will return 404
curl -u neo4j:password https://your-domain.com/
# Returns: "Not Found: This server only handles requests to /mcp"
```
