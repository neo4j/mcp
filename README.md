<!-- mcp-name: io.github.neo4j/mcp -->

# Neo4j MCP

Neo4j MCP gives AI assistants and LLM-powered tools direct, structured access to your Neo4j graph database.
By implementing the Model Context Protocol (MCP), it acts as a bridge between any MCP-compatible client, such as Claude, Cursor, or VS Code with MCP support, and your Neo4j instance.

## Features

- Explore your graph schema - discover node labels, relationship types, and property keys
- Let AI reason on your data model without prior knowledge
- Run Cypher queries - execute, read, and write queries against your database in response to natural language prompts
- Inspect and analyze data - retrieve nodes, relationships, and paths to answer questions, generate summaries, or feed data to other workflows

## Tools

- `get-schema` — introspect labels, relationship types, property keys
- `read-cypher` — execute read-only Cypher queries that do not modify database data, enforced via `EXPLAIN` and Neo4j's query-type classification. **Note:** custom procedures or functions incorrectly classified as read-only by Neo4j may bypass this check; ensuring correct classification is the responsibility of the procedure/function maintainer.
- `write-cypher` — execute write Cypher queries (disabled if `NEO4J_READ_ONLY=true`)
- `list-gds-procedures` — list available GDS procedures
- `vector-search` — semantic vector search over a Neo4j vector index. Embeds the query text at query time (via the Neo4j GenAI plugin) and returns the most similar nodes with similarity scores, with optional metadata filters. Only registered when an embedding provider is configured (see [Vector search configuration](#vector-search-configuration)).

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

## Vector search configuration

The `vector-search` tool is enabled only when an embedding provider is configured. The
server embeds the query text inside Neo4j using the GenAI plugin, so the target instance
must have the GenAI plugin available (standard on Neo4j Aura) and at least one vector
index. It uses the GenAI plugin's `ai.text.embed()` function on Neo4j 2025.11 and later,
and automatically falls back to the deprecated-but-present `genai.vector.encode()`
function on older versions — including the 5.x releases currently deployed on Neo4j Aura —
so no user action is needed. The embedding **model must match the model used to
create the stored embeddings**, otherwise similarity scores are meaningless.

Configure via environment variables:

| Variable | Applies to | Description |
|---|---|---|
| `NEO4J_EMBEDDING_PROVIDER` | all | `openai`, `azure-openai`, `vertexai`, or `bedrock-titan`. Empty disables `vector-search`. |
| `NEO4J_EMBEDDING_MODEL` | all | Embedding model id, e.g. `text-embedding-3-small`. |
| `NEO4J_EMBEDDING_API_KEY` | openai, azure-openai, vertexai | Provider API token. |
| `NEO4J_EMBEDDING_DIMENSIONS` | optional | Output dimensions, only if your model/index uses a reduced size. |
| `NEO4J_EMBEDDING_AZURE_RESOURCE` | azure-openai | Azure resource name. |
| `NEO4J_EMBEDDING_VERTEX_PROJECT` | vertexai | Google Cloud project id. |
| `NEO4J_EMBEDDING_VERTEX_REGION` | vertexai | GCP region. |
| `NEO4J_EMBEDDING_VERTEX_PUBLISHER` | vertexai | Optional, defaults to `google`. |
| `NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID` | bedrock-titan | AWS access key id. |
| `NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY` | bedrock-titan | AWS secret access key. |
| `NEO4J_EMBEDDING_AWS_REGION` | bedrock-titan | AWS region. |

The API key is read only from the server environment, is passed to Neo4j as a bound query
parameter (obfuscated in the query log), and is never accepted as a tool input or returned
to clients. In production deployments, source it from a secret manager rather than plain
text. Example (OpenAI) for an `mcp.json` `env` block:

```json
"NEO4J_EMBEDDING_PROVIDER": "openai",
"NEO4J_EMBEDDING_MODEL": "text-embedding-3-small",
"NEO4J_EMBEDDING_API_KEY": "sk-..."
```

## Links

- [Documentation](https://neo4j.com/docs/mcp/current/): The official Neo4j MCP documentation.
- [Discord](https://discord.gg/neo4j): The Neo4j discord channel.
- [Contributing Guide](CONTRIBUTING.md): Contribution workflow, development environment, mocks and testing.

For issues and feedback, you can also create a GitHub issue with reproduction details (omit sensitive data).
