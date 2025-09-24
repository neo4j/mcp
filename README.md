# Neo4j MCP (BETA)

Official Model Context Protocol (MCP) server for Neo4j.

## Status

BETA - Active development; not yet suitable for production.

## Prerequisites

- A running Neo4j database instance; either local [neo4j–desktop](https://neo4j.com/download/) or [Aura](https://neo4j.com/product/auradb/).
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
        "NEO4J_DATABASE": "neo4j"
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
        "NEO4J_DATABASE": "neo4j"
      }
    }
  }
}
```

Notes:

- Adjust env vars for your setup (defaults shown above).
- Neo4j Desktop default URI: `bolt://localhost:7687`.
- Aura: use the connection string from the Aura console.

## Tools & Usage

Provided tools:

| Tool         | Purpose                                              | Notes                                                                                      |
| ------------ | ---------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `get-schema` | Introspect labels, relationship types, property keys | Read-only. Provide valuable context to the client LLMs.                                    |
| `run-cypher` | Execute arbitrary Cypher (read/write)                | **Caution:** LLM-generated queries could cause harm. Use only in development environments. |

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
