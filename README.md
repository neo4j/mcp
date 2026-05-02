<!-- mcp-name: io.github.neo4j/mcp -->
# Neo4j MCP

Neo4j MCP gives AI assistants and LLM-powered tools direct, structured access to your Neo4j graph database.
By implementing the Model Context Protocol (MCP), it acts as a bridge between any MCP-compatible client, such as Claude, Cursor, or VS Code with MCP support, and your Neo4j instance.


## Features

* Explore your graph schema - discover node labels, relationship types, and property keys
* Let AI reason on your data model without prior knowledge
* Run Cypher queries - execute, read, and write queries against your database in response to natural language prompts
* Inspect and analyze data - retrieve nodes, relationships, and paths to answer questions, generate summaries, or feed data to other workflows


## Installation

**Install with PyPI:**

```bash
pip install neo4j-mcp-server
```

Otherwise see [MCP documentation -> Installation](https://neo4j.com/docs/mcp/current/installation).


## Server configuration (VSCode)

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

See [MCP documentation > Configuration](https://neo4j.com/docs/mcp/current/configuration) for more details.


## Links

- [Documentation](https://neo4j.com/docs/mcp/current/): The official Neo4j MCP documentation.
- [Discord](https://discord.gg/neo4j): The Neo4j discord channel.
- [Contributing Guide](CONTRIBUTING.md): Contribution workflow, development environment, mocks and testing.

For issues and feedback, you can also create a GitHub issue with reproduction details (omit sensitive data).