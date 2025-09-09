# Neo4j MCP

Official repository for the Neo4j MCP.

## Status

This project is currently under active development and is not ready for production use.

## Setup

### 1. Clone the Repository

```bash
git clone https://github.com/neo4j/mcp.git
cd mcp
```

### 2. Set up Go Environment

Ensure your Go environment is properly configured:

```bash
# Check Go installation
go version
```

Set up GOPATH and GOBIN (if not already configured in your favorite shell file, such as: .bashrc,.zshrc)

```bash
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN
```

### 3. Install the Neo4j MCP Server

```bash
go install -C cmd/neo4j-mcp
```

This will install the `neo4j-mcp` binary to your `$GOBIN` directory.

### 4. Configure VSCode MCP

Create or update your VSCode MCP configuration file (`mcp.json`), as document here:https://code.visualstudio.com/docs/copilot/customization/mcp-servers

```json
{
  "servers": {
    "neo4j": {
      "type": "stdio",
      "command": "./bin/neo4j-mcp", // Use full path to binary or ensure neo4j-mcp is in PATH
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

Adjust the environment variables according to your Neo4j instance configuration.

Open the VSCode chat in agentic mode and ask about your configured Neo4j database.

## Documentation

📚 **[Full Documentation](docs/README.md)** - Complete guides and architecture documentation

- [Database Interface Testing](docs/database-interfaces.md) - Guide to interface-based testing and mocking

## Extra

### Building and Running

To build the project:

```bash
go build -C cmd/neo4j-mcp -o ../../bin/
```

To run directly:

```bash
go run ./cmd/neo4j-mcp
```

To install:

```bash
go install -C cmd/neo4j-mcp
```

### Testing with MCP Inspector

Use the MCP Inspector to test and explore the functionality:

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```

This will launch the MCP Inspector interface where you can interact with the Neo4j MCP server and test the read-cypher capabilities.
