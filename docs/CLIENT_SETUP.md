# MCP Client Setup Guide

This guide covers how to configure various MCP clients (VSCode, Claude Desktop, etc.) to use the Neo4j MCP server.

The server supports two transport modes:

- **STDIO** (default): For desktop clients (Claude Desktop, VSCode)
- **HTTP**: For web-based clients and multi-tenant scenarios

See [README.md](../README.md#transport-modes) for more details on transport modes.

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
export NEO4J_TRANSPORT_MODE="http"
```

**Important:** Do NOT set `NEO4J_USERNAME` or `NEO4J_PASSWORD` for HTTP mode. Credentials come from per-request headers (Bearer token or Basic Auth).

**Authentication Methods:**

The HTTP mode supports two authentication methods:

1. **Bearer Token** (for Neo4j Enterprise/Aura with SSO/OAuth):

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer your-sso-token-here" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

2. **Basic Auth** (traditional username/password):

```bash
curl -X POST http://localhost:8080/mcp \
  -u neo4j:password \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

**Which Authentication Method Should I Use?**

- **Use Bearer Token** when:
  - You're using Neo4j Enterprise Edition or Aura with SSO/OIDC/OAuth configured
  - You want to integrate with your organization's identity provider
  - You need to support OAuth 2.0 flows

- **Use Basic Auth** when:
  - You're using traditional username/password authentication
  - You're using Neo4j Community Edition
  - You have direct database credentials

**Custom auth header name**

By default, the server reads credentials from the `Authorization` header.
You can change the header name the server reads from by setting the environment variable `NEO4J_HTTP_AUTH_HEADER_NAME`
or passing the CLI flag `--neo4j-http-auth-header-name` when starting `neo4j-mcp`.

Example (custom header `X-Test-Auth`):

```bash
export NEO4J_TRANSPORT_MODE="http"
export NEO4J_HTTP_AUTH_HEADER_NAME="X-Test-Auth"
neo4j-mcp
# Then send requests like:
# -H "X-Test-Auth: Bearer your-sso-token-here"
# or use -H "X-Test-Auth: Basic <base64>"
```

**Optional:**

```bash
# HTTP server configuration
export NEO4J_MCP_HTTP_HOST="127.0.0.1"      # Default: 127.0.0.1
export NEO4J_MCP_HTTP_PORT="80"             # Default: 80
export NEO4J_MCP_HTTP_ALLOWED_ORIGINS="*"   # Default: empty (no CORS)
export NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING="false" # Allow unauthenticated ping probes (default: false)

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
export NEO4j_TRANSPORT_MODE="http"
neo4j-mcp
```

The server will start on `http://127.0.0.1:80` by default.

Then create or edit your `mcp.json` file.

**Option 1: Basic Authentication**

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

**Option 2: Bearer Token (Enterprise/Aura with SSO)**

For Neo4j Enterprise or Aura with SSO/OAuth configured:

```json
{
  "servers": {
    "neo4j-http-bearer": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
      }
    }
  }
}
```

**Note:** Replace the bearer token with your actual OAuth/SSO token from your identity provider.

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

**Option 1: Basic Authentication**

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

**Option 2: Bearer Token (Enterprise/Aura with SSO)**

For Neo4j Enterprise or Aura with SSO/OAuth configured:

```json
{
  "mcpServers": {
    "neo4j-http-bearer": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
      }
    }
  }
}
```

**Note:** Replace the bearer token with your actual OAuth/SSO token from your identity provider.

## Amazon Kiro Configuration

The Amazon Kiro (https://kiro.dev/) is an agentic IDE that helps you do your best work with features such as specs, steering, and hooks.

The Kiro MCP configuration is managed through a JSON file. The interface supports two levels of configuration:

- User-level: ~/.kiro/settings/mcp.json applies to all workspaces
- Workspace-level: .kiro/settings/mcp.json is specific to the current workspace

You can apply the configuration using :

- Open the command palette (use Cmd + Shift + P on Mac or Ctrl + Shift + P on Windows/Linux), search for MCP and select Kiro: Open workspace MCP config (JSON) or Kiro: Open user MCP config (JSON), depending on whether or not you want to configure the settings for the workspace or user level.

### HTTP Mode

HTTP mode uses per-request authentication (Bearer Token or Basic Auth):

First, start your Neo4j MCP server in HTTP mode (see [HTTP Mode](#http-mode) section above).

Then edit your Kiro configuration file as mentioned above:

**Configuration:**

**Option 1: Basic Authentication**

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

**Option 2: Bearer Token (Enterprise/Aura with SSO)**

For Neo4j Enterprise or Aura with SSO/OAuth configured:

```json
{
  "mcpServers": {
    "neo4j-http-bearer": {
      "type": "http",
      "url": "http://127.0.0.1:80/mcp",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
      }
    }
  }
}
```

**Note:** Replace the bearer token with your actual OAuth/SSO token from your identity provider.

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

HTTP mode uses per-request authentication (Bearer Token or Basic Auth):

- **Required**: All HTTP requests must include authentication headers (Bearer or Basic)
- **Bearer Token Support**: Supports Neo4j Enterprise/Aura SSO/OAuth authentication
- **Basic Auth Support**: Traditional username/password authentication
- **Per-Request Credentials**: Each HTTP request uses its own Neo4j credentials
- **Multi-Tenant Support**: Different users can access different Neo4j databases/credentials
- **No Shared State**: HTTP mode is stateless - credentials never stored on server
- **Security**: Returns 401 if credentials are missing

The server uses Neo4j's impersonation feature to execute queries with different credentials without creating new driver instances (more efficient).

## Troubleshooting Authentication

### Bearer Token Issues

**401 Unauthorized - Token not accepted**
- Verify your Neo4j instance is Enterprise Edition or Aura with OAuth/SSO configured
- Community Edition does not support bearer token authentication - use Basic Auth instead
- Confirm the token hasn't expired - bearer tokens typically have short lifespans (15-60 minutes)
- Check with your identity provider to ensure the token is valid

**Invalid token format**
- Ensure the header format is exactly: `Authorization: Bearer YOUR_TOKEN`
- No extra spaces before or after "Bearer"
- Token should not be base64-encoded (unlike Basic Auth)

**Neo4j rejects valid token**
- Verify your Neo4j instance is configured to accept tokens from your identity provider
- Check Neo4j server logs for specific authentication errors
- Confirm the token issuer matches Neo4j's OAuth configuration

### Basic Auth Issues

**401 Unauthorized - Credentials not accepted**
- Verify username and password are correct
- Check that credentials are properly base64 encoded: `echo -n "user:pass" | base64`
- Ensure the authorization header format is: `Authorization: Basic BASE64_STRING`

**Empty credentials error**
- Both username and password must be non-empty
- The base64 string must decode to `username:password` format with both parts present

### General Issues

**No authentication header provided**
- HTTP mode requires authentication on every request
- Ensure your MCP client configuration includes the `Authorization` header

**Connection refused / Cannot reach server**
- Verify the Neo4j MCP server is running in HTTP mode
- Check the server is listening on the correct host:port
- Confirm firewall rules allow connections to the MCP server port

## Additional Clients

Configuration instructions for other MCP clients will be added here as they become available.

## Need Help?

- Check the main [README](../README.md) for general information
- See [CONTRIBUTING](../CONTRIBUTING.md) for development and testing
- Open an issue at https://github.com/neo4j/mcp/issues
