---
name: Bug report
about: Create a report to help us improve
title: ''
labels: 'bug'
assignees: ''

---

**Description:**

A clear and concise description of what the bug is.

**How to Reproduce:**

The most effective way to report a bug is by providing the JSON-RPC request and response. You can often get this from the MCP Inspector.

_Request_:

```json
{
  "jsonrpc": "2.0",
  "method": "...",
  "id": 1,
  "params": {
    ...
  }
}
```

_Response_:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    ...
  }
}
```

**Alternatively, provide manual steps:**

If you cannot provide the JSON-RPC data, please list the manual steps to reproduce the bug. Client-specific instructions (e.g., steps in VS Code Copilot, Cursor, Claude Desktop) are welcome.

1.  Go to '...'
2.  Click on '....'
3.  Scroll down to '....'
4.  See error

**Expected Behavior:**

A clear and concise description of what you expected to happen.

**Actual Behavior:**

A clear and concise description of what actually happened. Please include the full error message if available.

**Client:**

e.g., VS Code Copilot, Cursor, Claude Desktop

**MCP Version:**

e.g., 0.3.0

**Database State (if needed):**

Please describe the state of the database needed to reproduce the issue.
e.g., Empty, or with the following data loaded: [...]

**Database Info (if applicable):**

Please provide the database version and any relevant plugins.
e.g., Neo4j 5.11, APOC 5.11.0, GDS 2.5.3

**OS/Arc:**

e.g., MacOS/ARM, Linux/x64
