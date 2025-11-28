package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// go run ./client/... bin/neo4j-mcp
func main() {
	log.Print("Starting server...")
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	// Initialize the client
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <path_to_mcp_program>")
	}
	program := os.Args[1]
	log.Print(program)
	c, err := client.NewStdioMCPClient(
		program,
		os.Environ(), // passthrough environments
		os.Args[2:]...,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()
	captureServerLog(c)

	fmt.Println("Initializing client...")
	// Start the client
	if err := c.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}

	serverInfo, err := c.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	fmt.Printf(
		"Initialized with server: %s %s\n\n",
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version,
	)
	// Perform health check using ping
	fmt.Println("Performing health check...")
	if err := c.Ping(ctx); err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Println("Server is alive and responding")

	// List available tools if the server supports them
	if serverInfo.Capabilities.Tools != nil {
		fmt.Println("Fetching available tools...")
		toolsRequest := mcp.ListToolsRequest{}
		toolsResult, err := c.ListTools(ctx, toolsRequest)
		if err != nil {
			log.Fatalf("Failed to list tools: %v", err)
		}
		fmt.Printf("Server has %d tools available\n", len(toolsResult.Tools))
		for i, tool := range toolsResult.Tools {
			fmt.Printf("  %d. %s - %s\n", i+1, tool.Name, tool.Description)
		}
	}

	// List available resources if the server supports them
	if serverInfo.Capabilities.Resources != nil {
		fmt.Println("Fetching available resources...")
		resourcesRequest := mcp.ListResourcesRequest{}
		resourcesResult, err := c.ListResources(ctx, resourcesRequest)
		if err != nil {
			log.Fatalf("Failed to list resources: %v", err)
		}
		fmt.Printf("Server has %d resources available\n", len(resourcesResult.Resources))
		for i, resource := range resourcesResult.Resources {
			fmt.Printf("  %d. %s - %s\n", i+1, resource.URI, resource.Name)
		}
	}
	callGetSchema(context.Background(), c)
	callReadCypher(context.Background(), c)
	fmt.Println("Client initialized successfully. Shutting down...")
}

func callReadCypher(ctx context.Context, c *client.Client) {
	// Call the read-cypher tool to retrieve Neo4j database schema
	fmt.Println("Calling read-cypher tool...")
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "read-cypher",
			Arguments: map[string]interface{}{
				"Query": "RETURN 1",
			},
		},
	}

	toolResult, err := c.CallTool(ctx, toolRequest)
	if err != nil {
		log.Fatalf("Failed to call read-cypher tool: %v", err)
	}

	fmt.Printf("Schema result: %+v\n", toolResult)
	if toolResult.IsError == true {
		log.Fatal("error with Tool", toolResult.Content)
	}

}

func callGetSchema(ctx context.Context, c *client.Client) {
	// Call the get-schema tool to retrieve Neo4j database schema
	fmt.Println("Calling get-schema tool...")
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get-schema",
			Arguments: map[string]interface{}{},
		},
	}

	toolResult, err := c.CallTool(ctx, toolRequest)
	if err != nil {
		log.Fatalf("Failed to call get-schema tool: %v", err)
	}

	fmt.Printf("Schema result: %+v\n", toolResult)
	if toolResult.IsError == true {
		log.Fatal("error with Tool", toolResult.Content)
	}

}

func captureServerLog(c *client.Client) {
	// Set up logging for stderr if available
	if stderr, ok := client.GetStderr(c); ok {
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stderr.Read(buf)
				if err != nil {

					if err != io.EOF {
						log.Printf("Error reading stderr: %v", err)
					}
					return
				}
				if n > 0 {
					fmt.Fprintf(os.Stderr, "[Server] %s", buf[:n])
				}
			}
		}()
	}
}
