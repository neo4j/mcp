//go:build e2e

package helpers

import "github.com/mark3labs/mcp-go/mcp"

func BuildInitializeRequest() mcp.InitializeRequest {
	InitializeRequest := mcp.InitializeRequest{}
	InitializeRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	InitializeRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
	return InitializeRequest
}
