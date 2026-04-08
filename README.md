# Neo4j MCP

Official Model Context Protocol (MCP) server for Neo4j.

## Links

- [Documentation](https://neo4j.com/docs/mcp/current/)
- [Discord](https://discord.gg/neo4j)
- [Client Setup Guide](docs/CLIENT_SETUP.md)onfigure VSCode, Claude Desktop, and other MCP clients (STDIO and HTTP modes)
- [Contributing Guide](CONTRIBUTING.md): Contribution workflow, development environment, mocks & testing

For issues and feedback, create a GitHub issue with reproduction details (omit sensitive data).


## Startup Checks & Adaptive Operation

The server performs several pre-flight checks at startup to ensure your environment is correctly configured.

**STDIO Mode - Mandatory Requirements**
In STDIO mode, the server verifies the following core requirements. If any of these checks fail (e.g., due to an invalid configuration, incorrect credentials, or a missing APOC installation), the server will not start:

- A valid connection to your Neo4j instance.
- The ability to execute queries.
- The presence of the APOC plugin.

**HTTP Mode - Verification Skipped**
In HTTP mode, startup verification checks are skipped because credentials come from per-request Basic Auth headers. The server starts immediately without connecting to Neo4j at startup.

**Optional Requirements**
If an optional dependency is missing, the server will start in an adaptive mode. For instance, if the Graph Data Science (GDS) library is not detected in your Neo4j installation, the server will still launch but will automatically disable all GDS-related tools, such as `list-gds-procedures`. All other tools will remain available.


See the [Client Setup Guide](docs/CLIENT_SETUP.md) for configuration instructions for both modes.

## Unauthenticated MCP Ping

By default, the ping method is protected by standard authentication flows. However, because the MCP specification allows pings prior to initialization, some integrations (such as AWS AgentCore) rely on this optional method as an initial health check mechanism.

To improve integration compatibility with these platforms, you can exclude the ping method from authentication requirements via:

- Environment Variable: `NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING=true`
- or Command Line Flag: `--neo4j-http-allow-unauthenticated-ping true`

## TLS/HTTPS Configuration

When using HTTP transport mode, you can enable TLS/HTTPS for secure communication:

### Environment Variables

- `NEO4J_MCP_HTTP_TLS_ENABLED` - Enable TLS/HTTPS: `true` or `false` (default: `false`)
- `NEO4J_MCP_HTTP_TLS_CERT_FILE` - Path to TLS certificate file (required when TLS is enabled)
- `NEO4J_MCP_HTTP_TLS_KEY_FILE` - Path to TLS private key file (required when TLS is enabled)
- `NEO4J_MCP_HTTP_PORT` - HTTP server port (default: `443` when TLS enabled, `80` when TLS disabled)
- `NEO4J_HTTP_AUTH_HEADER_NAME` - Name of the HTTP header to read auth credentials from (default: `Authorization`)
- `NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING` - Allow unauthenticated ping health checks (default: `false`)

### Security Configuration

- **Minimum TLS Version**: Hardcoded to TLS 1.2 (allows TLS 1.3 negotiation)
- **Cipher Suites**: Uses Go's secure default cipher suites
- **Default Port**: Automatically uses port 443 when TLS is enabled (standard HTTPS port)

### Example Configuration

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_TRANSPORT_MODE="http"
export NEO4J_MCP_HTTP_TLS_ENABLED="true"
export NEO4J_MCP_HTTP_TLS_CERT_FILE="/path/to/cert.pem"
export NEO4J_MCP_HTTP_TLS_KEY_FILE="/path/to/key.pem"

neo4j-mcp
# Server will listen on https://127.0.0.1:443 by default
```

**Production Usage**: Use certificates from a trusted Certificate Authority (e.g., Let's Encrypt, or your organisation) for production deployments.

For detailed instructions on certificate generation, testing TLS, and production deployment, see [CONTRIBUTING.md](CONTRIBUTING.md#tlshttps-configuration).

## Configuration Options

The `neo4j-mcp` server can be configured using environment variables or CLI flags. CLI flags take precedence over environment variables.

### Environment Variables

See the [Client Setup Guide](docs/CLIENT_SETUP.md) for configuration examples.

### CLI Flags

You can override any environment variable using CLI flags:

```bash
neo4j-mcp --neo4j-uri "bolt://localhost:7687" \
          --neo4j-username "neo4j" \
          --neo4j-password "password" \
          --neo4j-database "neo4j" \
          --neo4j-read-only false \
          --neo4j-telemetry true
```

Available flags:

- `--neo4j-uri` - Neo4j connection URI (overrides NEO4J_URI)
- `--neo4j-username` - Database username (overrides NEO4J_USERNAME)
- `--neo4j-password` - Database password (overrides NEO4J_PASSWORD)
- `--neo4j-database` - Database name (overrides NEO4J_DATABASE)
- `--neo4j-read-only` - Enable read-only mode: `true` or `false` (overrides NEO4J_READ_ONLY)
- `--neo4j-telemetry` - Enable telemetry: `true` or `false` (overrides NEO4J_TELEMETRY)
- `--neo4j-schema-sample-size` - Modify the sample size used to infer the Neo4j schema
- `--neo4j-transport-mode` - Transport mode: `stdio` or `http` (overrides NEO4J_TRANSPORT_MODE)
- `--neo4j-http-host` - HTTP server host (overrides NEO4J_MCP_HTTP_HOST)
- `--neo4j-http-port` - HTTP server port (overrides NEO4J_MCP_HTTP_PORT)
- `--neo4j-http-tls-enabled` - Enable TLS/HTTPS: `true` or `false` (overrides NEO4J_MCP_HTTP_TLS_ENABLED)
- `--neo4j-http-tls-cert-file` - Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE)
- `--neo4j-http-tls-key-file` - Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE)
- `--neo4j-http-auth-header-name` - Name of the HTTP header to read auth credentials from (overrides NEO4J_AUTH_HEADER_NAME)
- `--neo4j-http-allow-unauthenticated-ping` - Allow unauthenticated ping health checks: `true` or `false` (overrides NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING)

Use `neo4j-mcp --help` to see all available options.


## Client Configuration

To configure MCP clients (VSCode, Claude Desktop, etc.) to use the Neo4j MCP server, see:

📘 **[Client Setup Guide](docs/CLIENT_SETUP.md)** – Complete configuration for STDIO and HTTP modes


## Telemetry

By default, `neo4j-mcp` collects anonymous usage data to help us improve the product.
This includes information like the tools being used, the operating system, and CPU architecture.
We do not collect any personal or sensitive information.

To disable telemetry, set the `NEO4J_TELEMETRY` environment variable to `"false"`. Accepted values are `true` or `false` (default: `true`).

You can also use the `--neo4j-telemetry` CLI flag to override this setting.
