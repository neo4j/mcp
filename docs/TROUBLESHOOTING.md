# Neo4j MCP Troubleshooting Guide

This guide covers common issues and debugging steps for the Neo4j MCP server.

## Common Issues

### Connection Problems

#### Connection Failed

**Symptoms:** MCP server cannot connect to Neo4j database
**Solutions:**

- Verify your Neo4j instance is running and accessible at the configured URI
- Check if the port (default 7687 for bolt) is open and not blocked by firewall
- For Neo4j Desktop: ensure the database is started
- For Neo4j Aura: verify your connection URI from the Aura console

#### Authentication Error

**Symptoms:** "Authentication failed" or similar credential errors
**Solutions:**

- Check your username and password in the MCP configuration
- Verify credentials by connecting with Neo4j Browser or Cypher Shell
- For Neo4j Desktop: use the password you set during database creation
- For Neo4j Aura: use the credentials provided when the database was created

#### SSL/TLS Issues

**Symptoms:** SSL handshake failures or certificate errors
**Solutions:**

- For local development: use `bolt://` instead of `bolt+s://`
- For Neo4j Aura: ensure you're using `neo4j+s://` or `bolt+s://`
- Check if your firewall or proxy is interfering with SSL connections

### Configuration Issues

#### Binary Not Found

**Symptoms:** "Command not found" or "No such file or directory"
**Solutions:**

- Verify the `neo4j-mcp` binary path in your MCP configuration is correct
- Use absolute path: `/full/path/to/neo4j-mcp` instead of relative path
- Ensure the binary is executable: `chmod +x /path/to/neo4j-mcp`
- Check if the binary is in your system PATH

#### Environment Variables Not Set

**Symptoms:** Connection attempts with default/empty values
**Solutions:**

- Verify all required environment variables are set in your MCP configuration:
  - `NEO4J_URI`
  - `NEO4J_USERNAME`
  - `NEO4J_PASSWORD`
  - `NEO4J_DATABASE`
- Check for typos in environment variable names
- Ensure values don't contain extra quotes or spaces

### Permission Issues

#### Permission Denied

**Symptoms:** "Access denied" or "Insufficient privileges"
**Solutions:**

- Ensure your Neo4j user has the necessary permissions for the operations you're trying to perform
- For read-only operations: user needs read access to the database
- For write operations: user needs write access to the database
- Check Neo4j's built-in roles: `reader`, `editor`, `publisher`, `architect`, `admin`

#### Database Access Denied

**Symptoms:** Cannot access specific database
**Solutions:**

- Verify the database name in your configuration matches the actual database
- Ensure the user has access to the specified database
- For multi-database setups: check database-specific permissions

### APOC Plugin Issues

#### APOC Procedures Not Found

**Symptoms:** "Procedure not found" errors when using schema introspection
**Solutions:**

- Install APOC plugin through Neo4j Desktop: Database → Plugins → APOC → Install
- For Neo4j Server: download and install APOC manually
- Restart Neo4j after APOC installation
- Verify installation: run `CALL apoc.help()` in Neo4j Browser

#### APOC Version Compatibility

**Symptoms:** APOC procedures fail or return unexpected results
**Solutions:**

- Ensure APOC version is compatible with your Neo4j version
- Check [APOC compatibility matrix](https://github.com/neo4j-contrib/neo4j-apoc-procedures#version-compatibility-matrix)
- Update APOC to the correct version if needed

## Debugging Steps

### Enable Debug Logging

1. **Check VSCode Output Panel:**

   - Open VSCode → View → Output
   - Select "Model Context Protocol" from the dropdown
   - Look for connection and error messages

2. **Check Neo4j Logs:**
   - Neo4j Desktop: Database → Manage → Logs
   - Neo4j Server: Check `logs/` directory in Neo4j installation
   - Look for authentication and connection errors

### Test Connectivity

#### Using MCP Inspector

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```

This provides an interactive interface to test MCP server functionality.

#### Direct Connection Test

Test your Neo4j connection outside of MCP:

```bash
# Using cypher-shell (if installed)
cypher-shell -a bolt://localhost:7687 -u neo4j -p password

# Using curl (for HTTP API)
curl -X POST http://localhost:7474/db/data/cypher \
  -H "Content-Type: application/json" \
  -u neo4j:password \
  -d '{"query": "MATCH (n) RETURN count(n)"}'
```

#### Test Binary Execution

```bash
# Test if binary runs
./bin/neo4j-mcp --help

# Test with environment variables
NEO4J_URI=bolt://localhost:7687 \
NEO4J_USERNAME=neo4j \
NEO4J_PASSWORD=password \
NEO4J_DATABASE=neo4j \
./bin/neo4j-mcp
```

### Common Error Messages

#### "dial tcp: connection refused"

- Neo4j is not running
- Wrong port number in URI
- Firewall blocking connection

#### "authentication failed"

- Incorrect username/password
- User doesn't exist
- Password has expired (check Neo4j password policy)

#### "database does not exist"

- Wrong database name in configuration
- Database hasn't been created
- User doesn't have access to the specified database

#### "procedure not found: apoc.\*"

- APOC plugin not installed
- APOC plugin not compatible with Neo4j version
- Neo4j needs restart after APOC installation

## Getting Help

If you're still experiencing issues:

1. **Check the [Neo4j MCP GitHub Issues](https://github.com/neo4j/mcp/issues)** for similar problems
2. **Create a new issue** with:
   - Your configuration (without sensitive credentials)
   - Error messages from logs
   - Neo4j version and setup type (Desktop/Aura/Server)
   - Operating system
3. **Neo4j Community**: [Neo4j Community Forum](https://community.neo4j.com/)
4. **General MCP Issues**: [Model Context Protocol Documentation](https://modelcontextprotocol.io/)

## Useful Commands

```bash
# Test Go installation
go version

# Check if ports are open
nc -zv localhost 7687  # Test Neo4j bolt port
nc -zv localhost 7474  # Test Neo4j HTTP port

# Check running processes
ps aux | grep neo4j
```
