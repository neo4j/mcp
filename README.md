# Neo4j MCP (BETA)

Official Model Context Protocol (MCP) server for Neo4j.

## Status

BETA - Active development; not yet suitable for production.

## Prerequisites

- A running Neo4j database instance; options include [Aura](https://neo4j.com/product/auradb/), [neo4j–desktop](https://neo4j.com/download/) or [self-managed](https://neo4j.com/deployment-center/#gdb-tab).
- APOC plugin installed in the Neo4j instance.
- Any MCP-compatible client (e.g. [VSCode](https://code.visualstudio.com/) with [MCP support](https://code.visualstudio.com/docs/copilot/customization/mcp-servers))

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

## Configure VSCode (MCP)

Create / edit `mcp.json` (docs: https://code.visualstudio.com/docs/copilot/customization/mcp-servers):

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
        "NEO4J_READ_ONLY": "true" // Optional: disables write tools
      }
    }
  }
}
```

Restart VSCode; open Copilot Chat and ask: "List Neo4j MCP tools" to confirm.

## Configure Claude Desktop

First, make sure you have Claude for Desktop installed. [You can install the latest version here](https://claude.ai/download).

We’ll need to configure Claude for Desktop for whichever MCP servers you want to use. To do this, open your Claude for Desktop App configuration at:

- (MacOS/Linux) `~/Library/Application Support/Claude/claude_desktop_config.json`
- (Windows) `$env:AppData\Claude\claude_desktop_config.json`

in a text editor. Make sure to create the file if it doesn’t exist.

You’ll then add the `neo4j-mcp` MCP in the mcpServers key:

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
        "NEO4J_READ_ONLY": "true" // Optional: disables write tools
      }
    }
  }
}
```

Notes:

- Adjust env vars for your setup (defaults shown above).
- Set `NEO4J_READ_ONLY=true` to disable all write tools (e.g., `write-cypher`).
- When enabled, only read operations are available; write tools are not exposed to clients.
- Neo4j Desktop default URI: `bolt://localhost:7687`.
- Aura: use the connection string from the Aura console.

## Tools & Usage

Provided tools:

| Tool                  | ReadOnly | Purpose                                              | Notes                                                                                                                          |
| --------------------- | -------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `get-schema`          | `true`   | Introspect labels, relationship types, property keys | Provide valuable context to the client LLMs.                                                                                   |
| `read-cypher`         | `true`   | Execute arbitrary Cypher (read mode)                 | Rejects writes, schema/admin operations, and PROFILE queries. Use `write-cypher` instead.                                      |
| `write-cypher`        | `false`  | Execute arbitrary Cypher (write mode)                | **Caution:** LLM-generated queries could cause harm. Use only in development environments. Disabled if `NEO4J_READ_ONLY=true`. |
| `list-gds-procedures` | `true`   | List GDS procedures available in the Neo4j instance  | Help the client LLM to have a better visibility on the GDS procedures available                                                |

### Readonly mode flag

Enable readonly mode by setting the `NEO4J_READ_ONLY` environment variable to `true` (for example, `"NEO4J_READ_ONLY": "true"`).
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

## Documentation

📚 **[Contributing Guide](CONTRIBUTING.md)** – Contribution workflow, development environment, mocks & testing.

Issues / feedback: open a GitHub issue with reproduction details (omit sensitive data).
