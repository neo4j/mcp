## Build MCPB Bundle (for Claude Desktop)

You can package the MCP server into an `.mcpb` bundle for distribution with MCP clients (e.g., Anthropic/Claude).
see more details here: https://github.com/modelcontextprotocol/mcpb/.

### Important: Bundle configuration files

The final `.mcpb` build depends on:

- `manifest.json`
- `.mcpbignore`

```bash
# Install MCPB CLI (once)
npm install -g @anthropic-ai/mcpb

# build binaries for your OS/Architecture
go build -C cmd/neo4j-mcp -o ../../bin/

# Build bundle from the repository root with a custom name
mcpb pack . neo4j-official-mcp-1.0.0.mcpb
```

You can now go to your Claude Desktop and install the bundle just created.

On Claude Desktop:

- Open Settings page.
- Under Desktop app, click on the Extensions.
- Advanced settings.
- You can now install the extension using the button: "Install Extension"

Notes:

- Review `manifest.json` and `.mcpbignore` before packing.
- A limitation is present where it's not possible to override the command depending on the current Architecture (ARM/AMD64), this makes building a general purpose bundle for all the supported Architectures/OS less doable.
