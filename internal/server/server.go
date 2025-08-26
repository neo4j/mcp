package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-mcp/internal/config"
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	mcpServer *server.MCPServer
	config    *config.Config
	driver    *neo4j.DriverWithContext
}

// NewNeo4jMCPServer creates a new MCP server instance
func NewNeo4jMCPServer() *Neo4jMCPServer {
	mcpServer := server.NewMCPServer(
		"neo4-mcp",
		"0.0.1",
	)

	cfg := config.LoadConfig()

	// Initialize Neo4j driver once
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		log.Fatalf("Error creating Neo4j driver: %v\n", err)
	}

	return &Neo4jMCPServer{
		mcpServer: mcpServer,
		config:    cfg,
		driver:    &driver,
	}
}

// Define ReadCypher input schema
type ReadCypherInput struct {
	Query  string         `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params map[string]any `json:"params" jsonschema:"description=Parameters to pass to the Cypher query"`
}

// RegisterTools registers all available MCP tools
func (s *Neo4jMCPServer) RegisterTools() {
	// Register the read-cypher tool
	readCypher := mcp.NewTool("read-cypher",
		mcp.WithDescription("Perform a read-only Cypher against a Neo4j database"),
		mcp.WithInputSchema[ReadCypherInput](),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	s.mcpServer.AddTool(readCypher, s.handleReadCypher)
}

// Start starts the MCP server with stdio transport
func (s *Neo4jMCPServer) Start() error {
	fmt.Fprintf(os.Stderr, "Starting Cypher MCP Server running on stdio\n")

	// Start the server with stdio transport (this is blocking)
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Stop gracefully stops the server and closes the driver
func (s *Neo4jMCPServer) Stop() error {
	fmt.Print("Gracefully stop the server")
	ctx := context.Background()
	return (*s.driver).Close(ctx)
}

// handleReadCypher handles the read-cypher tool requests
func (s *Neo4jMCPServer) handleReadCypher(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args ReadCypherInput
	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params
	// TODO: remove these logs during productionization
	fmt.Fprintf(os.Stderr, "cypher-query: %s\n", Query)
	if Params != nil {
		fmt.Fprintf(os.Stderr, "cypher-parameters: %v\n", Params)
	}
	fmt.Fprintf(os.Stderr, "url: %s\n", s.config.URI)
	fmt.Fprintf(os.Stderr, "database: %s\n", s.config.Database)

	// Execute the Cypher query using the stored driver
	response, err := s.executeReadQuery(ctx, Query, Params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error executing Cypher query: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// executeReadQuery executes a read-only Cypher query and returns JSON-formatted results
func (s *Neo4jMCPServer) executeReadQuery(ctx context.Context, cypher string, params map[string]any) (string, error) {

	res, err := neo4j.ExecuteQuery(ctx, *s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(s.config.Database), neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing Cypher Read: %v\n", err)
		return "", err
	}

	var results []map[string]any
	for _, record := range res.Records {
		recordMap := make(map[string]any)
		for i, key := range record.Keys {
			recordMap[key] = record.Values[i]
		}
		results = append(results, recordMap)
	}

	formattedResponse, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting response as JSON: %v\n", err)
		return "", err
	}

	formattedResponseStr := string(formattedResponse)
	fmt.Fprintf(os.Stderr, "The formatted response: %s\n", formattedResponseStr)

	return formattedResponseStr, nil
}
