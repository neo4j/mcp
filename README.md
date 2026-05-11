<!-- mcp-name: io.github.neo4j/mcp -->

# Neo4j MCP

Neo4j MCP gives AI assistants and LLM-powered tools direct, structured access to your Neo4j graph database.
By implementing the Model Context Protocol (MCP), it acts as a bridge between any MCP-compatible client, such as Claude, Cursor, or VS Code with MCP support, and your Neo4j instance.

## Features

- Explore your graph schema - discover node labels, relationship types, and property keys
- Let AI reason on your data model without prior knowledge
- Run Cypher queries - execute, read, and write queries against your database in response to natural language prompts
- Inspect and analyze data - retrieve nodes, relationships, and paths to answer questions, generate summaries, or feed data to other workflows

## Migrating from v1 to v2

v2 makes HTTP mode multi-tenant: the target database and Neo4j URI now come from each request, so one server can serve multiple instances.

**STDIO mode** — `NEO4J_DATABASE` no longer defaults to `"neo4j"` and must be set explicitly. Otherwise the server fails to start with:

> `Neo4j database is required for STDIO mode (set NEO4J_DATABASE or use --neo4j-database flag)`

**HTTP mode** — server config moves to per-request headers and URL path.

Server environment:

```diff
- NEO4J_URI=bolt://host:7687
- NEO4J_DATABASE=neo4j
  NEO4J_TRANSPORT_MODE=http
```

Request:

```diff
- POST /mcp
+ POST /db/neo4j/mcp
+ X-Neo4j-MCP-URI: bolt://host:7687
  Authorization: Basic <base64>
```

### Breaking changes in HTTP mode to be aware of when migrating:

- If `NEO4J_DATABASE` is still set in HTTP mode, the server refuses to start with `NEO4J_DATABASE … should not be set for HTTP transport mode; database is selected per-request via URL path`.
- Similarly, setting `NEO4J_USERNAME` or `NEO4J_PASSWORD` in HTTP mode causes startup failure, since credentials must be provided via Basic Auth on each request.
- `NEO4J_URI` must also be empty at startup in HTTP mode, since the target Neo4j instance is determined per-request via the `X-Neo4j-MCP-URI` header. Setting `NEO4J_URI` in HTTP mode results in startup failure with `Neo4j URI should not be set for HTTP transport mode; URI is provided per-request via X-Neo4j-MCP-URI header`.

## Installation

**Install with PyPI:**

```bash
pip install neo4j-mcp-server
```

For manual installation or using `brew`, please see [MCP documentation -> Installation](https://neo4j.com/docs/mcp/current/installation).

## Server configuration (VSCode - STDIO)

Create / edit `mcp.json`:

```json
{
  "servers": {
    "neo4j": {
      "type": "stdio",
      "command": "python",
      "args": ["-m", "neo4j_mcp_server"],
      "env": {
        "NEO4J_URI": "bolt://localhost:7687",
        "NEO4J_USERNAME": "neo4j",
        "NEO4J_PASSWORD": "password",
        "NEO4J_DATABASE": "neo4j"
      }
    }
  }
}
```

See [MCP documentation > Configuration](https://neo4j.com/docs/mcp/current/configuration) for more details.

## Tools

- `get-schema` — introspect labels, relationship types, property keys
- `read-cypher` — execute read-only Cypher queries
- `write-cypher` — execute write Cypher queries (disabled if `NEO4J_READ_ONLY=true`)
- `list-gds-procedures` — list available GDS procedures

## Links

- [Documentation](https://neo4j.com/docs/mcp/current/): The official Neo4j MCP documentation.
- [Discord](https://discord.gg/neo4j): The Neo4j discord channel.
- [Contributing Guide](CONTRIBUTING.md): Contribution workflow, development environment, mocks and testing.

For issues and feedback, you can also create a GitHub issue with reproduction details (omit sensitive data).
