# Neo4j MCP

Official Model Context Protocol (MCP) server for Neo4j.

## Prerequisites

- A running Neo4j database instance; options include [Aura](https://neo4j.com/product/auradb/), [neo4j–desktop](https://neo4j.com/download/) or [self-managed](https://neo4j.com/deployment-center/#gdb-tab).
- APOC plugin installed in the Neo4j instance.
- Any MCP-compatible client (e.g. [VSCode](https://code.visualstudio.com/) with [MCP support](https://code.visualstudio.com/docs/copilot/customization/mcp-servers))

## Startup Checks & Adaptive Operation

The server performs several pre-flight checks at startup to ensure your environment is correctly configured.

**Mandatory Requirements**
The server verifies the following core requirements. If any of these checks fail (e.g., due to an invalid configuration, incorrect credentials, or a missing APOC installation), the server will not start:

- A valid connection to your Neo4j instance.
- The ability to execute queries.
- The presence of the APOC plugin.

**Optional Requirements**
If an optional dependency is missing, the server will start in an adaptive mode. For instance, if the Graph Data Science (GDS) library is not detected in your Neo4j installation, the server will still launch but will automatically disable all GDS-related tools, such as `list-gds-procedures`. All other tools will remain available.

## Installation (Binary)

Releases: https://github.com/neo4j/mcp/releases

1. Download the archive for your OS/arch.
2. Extract and place `neo4j-mcp` in a directory present in your PATH variables (see examples below).

Mac / Linux:

```bash
chmod +x neo4j-mcp
sudo mv neo4j-mcp /usr/local/bin/
```

Windows (PowerShell / cmd):

```powershell
move neo4j-mcp.exe C:\Windows\System32
```

Verify the neo4j-mcp installation:

```bash
neo4j-mcp -v
```

Should print the installed version.

## Configuration Options

The `neo4j-mcp` server can be configured using environment variables or CLI flags. CLI flags take precedence over environment variables.

### Environment Variables

See the configuration examples below for VSCode and Claude Desktop.

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

Use `neo4j-mcp --help` to see all available options.

## Configure VSCode (MCP)

Create / edit `mcp.json` (docs: https://code.visualstudio.com/docs/copilot/customization/mcp-servers):

```json
{
  "servers": {
    "neo4j": {
      "type": "stdio",
      "command": "neo4j-mcp",
      "env": {
        "NEO4J_URI": "bolt://localhost:7687", // Required: Neo4j connection URI
        "NEO4J_USERNAME": "neo4j", // Required: Database username
        "NEO4J_PASSWORD": "password", // Required: Database password
        "NEO4J_DATABASE": "neo4j", // Optional: Database name (default: neo4j)
        "NEO4J_READ_ONLY": "true", // Optional: Disables write tools (default: false)
        "NEO4J_TELEMETRY": "false", // Optional: Disables telemetry (default: true)
        "NEO4J_LOG_LEVEL": "info", // Optional: Log level (default: info)
        "NEO4J_LOG_FORMAT": "text", // Optional: Log format (default: text)
        "NEO4J_SCHEMA_SAMPLE_SIZE": "100" // Optional: Number of nodes to sample for schema inference (default: 100)
      }
    }
  }
}
```

**Note:** The first three environment variables (NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD) are **required**. The server will fail to start if any of these are missing.

Restart VSCode; open Copilot Chat and ask: "List Neo4j MCP tools" to confirm.

## Configure Claude Desktop

First, make sure you have Claude for Desktop installed. [You can install the latest version here](https://claude.ai/download).

We’ll need to configure Claude for Desktop for whichever MCP servers you want to use. To do this, open your Claude for Desktop App configuration at:

- (MacOS/Linux) `~/Library/Application Support/Claude/claude_desktop_config.json`
- (Windows) `$env:AppData\Claude\claude_desktop_config.json`

in a text editor. Make sure to create the file if it doesn’t exist.

You'll then add the `neo4j-mcp` MCP in the mcpServers key:

```json
{
  "mcpServers": {
    "neo4j-mcp": {
      "type": "stdio",
      "command": "neo4j-mcp",
      "args": [],
      "env": {
        "NEO4J_URI": "bolt://localhost:7687", // Required: Neo4j connection URI
        "NEO4J_USERNAME": "neo4j", // Required: Database username
        "NEO4J_PASSWORD": "password", // Required: Database password
        "NEO4J_DATABASE": "neo4j", // Optional: Database name (default: neo4j)
        "NEO4J_READ_ONLY": "true", // Optional: Disables write tools (default: false)
        "NEO4J_TELEMETRY": "false", // Optional: Disables telemetry (default: true)
        "NEO4J_LOG_LEVEL": "info", // Optional: Log level (default: info)
        "NEO4J_LOG_FORMAT": "text", // Optional: Log format (default: text)
        "NEO4J_SCHEMA_SAMPLE_SIZE": "100" // Optional: Number of nodes to sample for schema inference (default: 100)
      }
    }
  }
}
```

**Important Notes:**

- The first three environment variables (NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD) are **required**. The server will fail to start if any are missing.
- Neo4j Desktop default URI: `bolt://localhost:7687`
- Aura: use the connection string from the Aura console
- See the [Readonly Mode](#readonly-mode-flag), [Logging](#logging), and [Telemetry](#telemetry) sections below for more details on optional configuration.

## Tools & Usage

Provided tools:

| Tool                  | ReadOnly | Purpose                                              | Notes                                                                                                                          |
| --------------------- | -------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `get-schema`          | `true`   | Introspect labels, relationship types, property keys | Provide valuable context to the client LLMs.                                                                                   |
| `read-cypher`         | `true`   | Execute arbitrary Cypher (read mode)                 | Rejects writes, schema/admin operations, and PROFILE queries. Use `write-cypher` instead.                                      |
| `write-cypher`        | `false`  | Execute arbitrary Cypher (write mode)                | **Caution:** LLM-generated queries could cause harm. Use only in development environments. Disabled if `NEO4J_READ_ONLY=true`. |
| `list-gds-procedures` | `true`   | List GDS procedures available in the Neo4j instance  | Help the client LLM to have a better visibility on the GDS procedures available                                                |

### Readonly mode flag

Enable readonly mode by setting the `NEO4J_READ_ONLY` environment variable to `true` (for example, `"NEO4J_READ_ONLY": "true"`). Accepted values are `true` or `false` (default: `false`).

You can also override this setting using the `--neo4j-read-only` CLI flag:

```bash
neo4j-mcp --neo4j-uri "bolt://localhost:7687" --neo4j-username "neo4j" --neo4j-password "password" --neo4j-read-only true
```

When enabled, write tools (for example, `write-cypher`) are not exposed to clients.

### Query Classification

The `read-cypher` tool performs an extra round-trip to the Neo4j database to guarantee read-only operations.

Important notes:

- **Write operations**: `CREATE`, `MERGE`, `DELETE`, `SET`, etc., are treated as non-read queries.
- **Admin queries**: Commands like `SHOW USERS`, `SHOW DATABASES`, etc., are treated as non-read queries and must use `write-cypher` instead.
- **Profile queries**: `EXPLAIN PROFILE` queries are treated as non-read queries, even if the underlying statement is read-only.
- **Schema operations**: `CREATE INDEX`, `DROP CONSTRAINT`, etc., are treated as non-read queries.

## Example Natural Language Prompts

Below are some example prompts you can try in Copilot or any other MCP client:

- "What does my Neo4j instance contain? List all node labels, relationship types, and property keys."
- "Find all Person nodes and their relationships in my Neo4j instance."
- "Create a new User node with a name 'John' in my Neo4j instance."

## Security tips:

- Use a restricted Neo4j user for exploration.
- Review generated Cypher before executing in production databases.

## Logging

The server uses structured logging with support for multiple log levels and output formats.

### Configuration

**Log Level** (`NEO4J_LOG_LEVEL`, default: `info`)

Controls the verbosity of log output. Supports all [MCP log levels](https://modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging#log-levels): `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`.

**Log Format** (`NEO4J_LOG_FORMAT`, default: `text`)

Controls the output format:

- `text` - Human-readable text format (default)
- `json` - Structured JSON format (useful for log aggregation)

## Telemetry

By default, `neo4j-mcp` collects anonymous usage data to help us improve the product.
This includes information like the tools being used, the operating system, and CPU architecture.
We do not collect any personal or sensitive information.

To disable telemetry, set the `NEO4J_TELEMETRY` environment variable to `"false"`. Accepted values are `true` or `false` (default: `true`).

You can also use the `--neo4j-telemetry` CLI flag to override this setting.

## Documentation

📚 **[Contributing Guide](CONTRIBUTING.md)** – Contribution workflow, development environment, mocks & testing.

Issues / feedback: open a GitHub issue with reproduction details (omit sensitive data).
