# Neo4j MCP

Official Model Context Protocol (MCP) server for Neo4j.

## Status

Active development. Not yet production‑hardened.

## Prerequisites

- A running Neo4j database instance; either local [neo4j-desktop](https://neo4j.com/download/) or [Aura](https://neo4j.com/product/auradb/).
- Any MCP-compatible client (ie. [VSCode](https://code.visualstudio.com/) with [MCP support](https://code.visualstudio.com/docs/copilot/customization/mcp-servers))

## Installation (Binary)

Releases: https://github.com/neo4j/mcp/releases

1. Download the archive for your OS/arch.
2. Extract and place `neo4j-mcp` somewhere on your PATH (examples below).

Mac / Linux:

```bash
chmod +x neo4j-mcp
sudo mv neo4j-mcp /usr/local/bin/
```

Windows (PowerShell / cmd):

```powershell
move neo4j-mcp.exe C:\Windows\System32
```

To verify basic startup (will wait for MCP host on stdio):

```bash
neo4j-mcp # should block; Ctrl+C to stop
```

## Configure VSCode (MCP)

Create / edit `mcp.json` (docs: https://code.visualstudio.com/docs/copilot/customization/mcp-servers):

```json
{
  "servers": {
    "neo4j": {
      "type": "stdio",
      "command": "neo4j-mcp", // Use full path to binary or ensure neo4j-mcp is in PATH
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

<!-- TODO: add claude desktop MCP installation instructions -->

Notes:

- Adjust env vars for your setup (defaults shown above).
- Desktop default URI: `bolt://localhost:7687`.
- Aura: use the connection string from the Aura console.
- Provide an absolute path to the binary if not on PATH.

Restart VSCode; open Copilot Chat and ask: "List Neo4j MCP tools" to confirm.

## Tools & Usage

Provided tools:

| Tool         | Purpose                                                                    | Notes                                        |
| ------------ | -------------------------------------------------------------------------- | -------------------------------------------- |
| `get-schema` | Introspect labels, relationship types, property keys, indexes, constraints | Read-only, safe to repeat                    |
| `run-cypher` | Execute arbitrary Cypher (read/write)                                      | Use caution; respects Neo4j auth permissions |

Example natural language prompts (in Copilot / other MCP client):

```
Get the database schema
```

```
Find all Person nodes and their relationships
```

```
Create a new User node with name "John"
```

Security tips:

- Prefer a read-only Neo4j user for exploration.
- Review generated Cypher before executing in production databases.

## Documentation

📚 **[Contributing Guide](CONTRIBUTING.md)** – Contribution workflow, development environment, mocks & testing.

Issues / feedback: open a GitHub issue with reproduction details (omit sensitive data).
