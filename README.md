# Neo4j MCP

Official repository for the Neo4j MCP.

## Status

This project is currently under active development and is not ready for production use.

## Prerequisites

- A running Neo4j database instance; either local [neo4j-desktop](https://neo4j.com/download/) or [Aura](https://neo4j.com/product/auradb/).
- Any MCP-compatible client (ie. [VSCode](https://code.visualstudio.com/) with [MCP support](https://code.visualstudio.com/docs/copilot/customization/mcp-servers))

## Installation

The Neo4j MCP binaries are available on GitHub. Select the latest release and download the archive suitable for your platform and architecture of choice. The Neo4j MCP is fully compatible with Mac, Linux and Windows.

1. Navigate to https://github.com/neo4j/mcp/releases.
2. Download the compressed file that matches your OS. Make a note of the folder where the file is located.
3. Once the file is downloaded, extract the contents.
4. Open a command prompt and move to the location where you extracted the files.
5. Complete the installation by moving the neo4j-mcp executable file into the file path.

Mac/Linux users:

```bash
sudo mv neo4j-mcp /usr/local/bin
```

Windows users:

```bash
move neo4j-mcp c:\windows\system32
```

Check the installation with:

```bash
neo4j-mcp -v # todo: add a version flag to the binary
```

You should see the version of the Neo4j MCP displayed.

## Configure VSCode MCP

Create or update your VSCode MCP configuration file (`mcp.json`), as documented here: https://code.visualstudio.com/docs/copilot/customization/mcp-servers

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

**Configuration Notes:**

- Adjust the environment variables according to your Neo4j instance configuration (defaults are provided for Neo4j Desktop)
- For Neo4j Desktop: typically uses `bolt://localhost:7687` with your custom password
- For Neo4j Aura: use the connection URI provided in your Aura console
- Ensure the `neo4j-mcp` binary path is correct or the command is in your system PATH

After configuration, restart VSCode to load the MCP server. You can verify the connection by checking the MCP status in VSCode.

Open the VSCode chat in agentic mode and ask about your configured Neo4j database.

## MCP Usage

The Neo4j MCP server provides two main tools for interacting with your Neo4j database through Model Context Protocol:

### Available Tools

#### 1. `get-schema`

Retrieves comprehensive schema information from your Neo4j database, including:

- Node labels
- Relationship types
- Property keys and their data types
- Indexes and constraints

This tool is read-only and idempotent, making it safe to call repeatedly.

**Example usage:**

```
Get the database schema
```

#### 2. `run-cypher`

Executes Cypher queries against your Neo4j database. This tool supports:

- Read and write operations
- Parameterized queries for security
- Complex graph traversals and analytics

**Important:** This tool can perform destructive operations, so use with caution in production environments.

**Example usage:**

```
Find all Person nodes and their relationships
```

```
Create a new node with label User and name property "John"
```

```
Match all nodes connected to a specific node within 3 hops
```

### Integration with AI Assistants

Once configured, you can interact with your Neo4j database using natural language through supported AI assistants:

- **Schema Exploration**: "What's the structure of my database?" or "Show me all node types"
- **Data Querying**: "Find all users who purchased products in the last month"
- **Graph Analysis**: "Show me the shortest path between two specific nodes"
- **Data Modification**: "Create a new customer node with email and name"

The MCP server automatically translates your natural language requests into appropriate Cypher queries and executes them against your database.

### Security Considerations

- The `run-cypher` tool can execute any Cypher query, including destructive operations
- Ensure your Neo4j user has appropriate permissions for your use case
- Consider using a read-only user for exploration and analysis tasks
- Always validate results before applying changes in production environments

## Documentation

📚 **[Contributing Guide](CONTRIBUTING.md)** - How to contribute, development setup, coding standards, and technical architecture
