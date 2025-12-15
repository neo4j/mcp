# MCP Client Setup Guide

This guide covers how to configure various MCP clients (VSCode, Claude Desktop, etc.) to use the Neo4j MCP server in both STDIO and HTTP transport modes.

## Transport Modes Overview

The Neo4j MCP server supports two transport modes:

- **STDIO** (default): Standard MCP communication via stdin/stdout for desktop clients
- **HTTP**: RESTful HTTP server with per-request Basic Authentication for web-based clients and multi-tenant scenarios

### When to Use Each Mode

**Use STDIO mode when:**

- Using desktop MCP clients (Claude Desktop, VSCode)
- Single-user scenarios
- Local development
- Following standard MCP patterns

**Use HTTP mode when:**

- Building web-based MCP clients
- Need multi-tenant support (different users, different credentials)
- Want to expose MCP over network
- Integrating with web applications

## Environment Variables

### STDIO Mode

**Required:**

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
```

**Optional:**

```bash
export NEO4J_DATABASE="neo4j"               # Default: neo4j
export NEO4J_READ_ONLY="false"              # Default: false
export NEO4J_TELEMETRY="true"               # Default: true
export NEO4J_LOG_LEVEL="info"               # Default: info
export NEO4J_LOG_FORMAT="text"              # Default: text
export NEO4J_SCHEMA_SAMPLE_SIZE="100"       # Default: 100
```

### HTTP Mode

**Required:**

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_MCP_TRANSPORT="http"
```

**Important:** Do NOT set `NEO4J_USERNAME` or `NEO4J_PASSWORD` for HTTP mode. Credentials come from per-request Basic Auth headers.

**Optional:**

```bash
# HTTP server configuration
export NEO4J_MCP_HTTP_HOST="127.0.0.1"      # Default: 127.0.0.1
export NEO4J_MCP_HTTP_PORT="80"             # Default: 80
export NEO4J_MCP_HTTP_ALLOWED_ORIGINS="*"   # Default: empty (no CORS)

# Neo4j configuration (same as STDIO mode)
export NEO4J_DATABASE="neo4j"               # Default: neo4j
export NEO4J_READ_ONLY="false"              # Default: false
export NEO4J_TELEMETRY="true"               # Default: true
export NEO4J_LOG_LEVEL="info"               # Default: info
export NEO4J_LOG_FORMAT="text"              # Default: text
export NEO4J_SCHEMA_SAMPLE_SIZE="100"       # Default: 100
```

### CORS Configuration

The `NEO4J_MCP_HTTP_ALLOWED_ORIGINS` variable accepts:

- Empty string (default): CORS disabled
- `"*"`: Allow all origins
- Comma-separated list: `"http://localhost:3000,https://app.example.com"`

Example:

```bash
export NEO4J_MCP_HTTP_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:5173"
```

## VSCode Configuration

### STDIO Mode

Create or edit `mcp.json` (docs: https://code.visualstudio.com/docs/copilot/customization/mcp-servers):

```json
{
  "servers": {
    "neo4j": {
      "type": "stdio",
      "command": "neo4j-mcp",
      "env": {
        "NEO4J_URI": "bolt://localhost:7687",
        "NEO4J_USERNAME": "neo4j",
        "NEO4J_PASSWORD": "password",
        "NEO4J_DATABASE": "neo4j",
        "NEO4J_READ_ONLY": "true",
        "NEO4J_TELEMETRY": "false",
        "NEO4J_LOG_LEVEL": "info",
        "NEO4J_LOG_FORMAT": "text",
        "NEO4J_SCHEMA_SAMPLE_SIZE": "100"
      }
    }
  }
}
```

**Note:** The first three environment variables (NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD) are **required**. The server will fail to start if any of these are missing.

Restart VSCode; open Copilot Chat and ask: "List Neo4j MCP tools" to confirm.

### HTTP Mode

First, start your Neo4j MCP server in HTTP mode:

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_MCP_TRANSPORT="http"
neo4j-mcp
```

The server will start on `http://127.0.0.1:80` by default.

Then create or edit your `mcp.json` file:

```json
{
  "servers": {
    "neo4j-http": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Basic bmVvNGo6cGFzc3dvcmQ="
      }
    }
  }
}
```

**Generating the Authorization Header:**

The `Authorization` header value is `Basic` followed by base64-encoded `username:password`:

```bash
# On Mac/Linux
echo -n "neo4j:password" | base64
# Output: bmVvNGo6cGFzc3dvcmQ=

# Alternatively, you can use an online base64 encoder.
```

Then use it as: `"Authorization": "Basic bmVvNGo6cGFzc3dvcmQ="`

## Claude Desktop Configuration

### STDIO Mode

First, make sure you have Claude for Desktop installed. [You can install the latest version here](https://claude.ai/download).

Open your Claude for Desktop App configuration at:

- (MacOS/Linux) `~/Library/Application Support/Claude/claude_desktop_config.json`
- (Windows) `path_to_your\claude_desktop_config.json`

Create the file if it doesn't exist, then add the `neo4j-mcp` server:

```json
{
  "mcpServers": {
    "neo4j-mcp": {
      "type": "stdio",
      "command": "neo4j-mcp",
      "args": [],
      "env": {
        "NEO4J_URI": "bolt://localhost:7687",
        "NEO4J_USERNAME": "neo4j",
        "NEO4J_PASSWORD": "password",
        "NEO4J_DATABASE": "neo4j",
        "NEO4J_READ_ONLY": "true",
        "NEO4J_TELEMETRY": "false",
        "NEO4J_LOG_LEVEL": "info",
        "NEO4J_LOG_FORMAT": "text",
        "NEO4J_SCHEMA_SAMPLE_SIZE": "100"
      }
    }
  }
}
```

**Important Notes:**

- The first three environment variables (NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD) are **required**. The server will fail to start if any are missing.
- Neo4j Desktop default URI: `bolt://localhost:7687`
- Aura: use the connection string from the Aura console

### HTTP Mode

First, start your Neo4j MCP server in HTTP mode (see [HTTP Mode](#http-mode) section above).

Then edit your Claude Desktop configuration file:

**Location:**

- MacOS/Linux: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json` (your config location may vary)

**Configuration:**

```json
{
  "mcpServers": {
    "neo4j-http": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Basic bmVvNGo6cGFzc3dvcmQ="
      }
    }
  }
}
```

**Note:** Replace `bmVvNGo6cGFzc3dvcmQ=` with your own base64-encoded credentials (see [Generating the Authorization Header](#http-mode) section).

## Multi-User / Multi-Tenant Setup

HTTP mode supports multiple users with different credentials accessing the same server. You can configure multiple server entries with different credentials:

```json
{
  "mcpServers": {
    "neo4j-admin": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Basic YWRtaW46YWRtaW5wYXNz"
      }
    },
    "neo4j-readonly": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Basic cmVhZG9ubHk6cmVhZHBhc3M="
      }
    }
  }
}
```

Each server entry uses different Neo4j credentials, allowing you to switch between users in your MCP client. This is useful for:

- Testing with different permission levels
- Multi-tenant applications
- Switching between admin and read-only access

## Authentication

### STDIO Mode

Authentication is handled through environment variables (`NEO4J_USERNAME` and `NEO4J_PASSWORD`) that are configured when starting the server.

### HTTP Mode

HTTP mode uses per-request Basic Authentication:

- **Required**: All HTTP requests must include Basic Authentication headers
- **Per-Request Credentials**: Each HTTP request uses its own Neo4j credentials
- **Multi-Tenant Support**: Different users can access different Neo4j databases/credentials
- **No Shared State**: HTTP mode is stateless - credentials never stored on server
- **Security**: Returns 401 if credentials are missing

The server uses Neo4j's impersonation feature to execute queries with different credentials without creating new driver instances (more efficient).

## Additional Clients

Configuration instructions for other MCP clients will be added here as they become available.

## Need Help?

- Check the main [README](../README.md) for general information
- See [CONTRIBUTING](../CONTRIBUTING.md) for development and testing
- Open an issue at https://github.com/neo4j/mcp/issues
