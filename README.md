# Neo4j MCP

Neo4j MCP gives AI assistants and LLM-powered tools direct, structured access to your Neo4j graph database.
By implementing the Model Context Protocol (MCP), it acts as a bridge between any MCP-compatible client, such as Claude, Cursor, or VS Code with MCP support, and your Neo4j instance.

Neo4j MCP is intended for:

* Developers building or prototyping graph-backed AI applications who want to query Neo4j conversationally during development.
* Data scientists and analysts who want to explore graph data without deep Cypher expertise.
* Platform and infrastructure teams deploying shared AI tooling that needs structured, auditable access to a Neo4j instance.
* AI application builders integrating Neo4j as a knowledge source or reasoning backend in multi-agent systems.

Neo4j MCP enables AI agents to:

* Explore your graph schema - discover node labels, relationship types, and property keys so the AI can reason about your data model without prior knowledge of it.
* Run Cypher queries - execute, read, and write queries against your database in response to natural language prompts.
* Inspect and analyze data - retrieve nodes, relationships, and paths to answer questions, generate summaries, or feed data to other workflows.

## Links

- [Documentation](https://neo4j.com/docs/mcp/current/)
- [Discord](https://discord.gg/neo4j)
- [Contributing Guide](CONTRIBUTING.md): Contribution workflow, development environment, mocks & testing

For issues and feedback, create a GitHub issue with reproduction details (omit sensitive data).