// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/tools"
)

const (
	defaultTopK     = 5
	maxTopK         = 100
	overFetchFactor = 5
	maxFetchK       = 1000
)

// SearchHandler returns an MCP tool handler that performs semantic vector search.
//
//   - deps: tool dependencies (DBService must be non-nil at call time).
//   - emb: embedding provider configuration read from server env vars — never from tool input.
//   - useSearchClause: evaluated at call time — true → SEARCH clause (Neo4j ≥ 2026.01);
//     false → db.index.vector.queryNodes() fallback. It is a function (not a bool) so the
//     tool can be registered at startup, before the Neo4j version has been detected: in
//     HTTP mode the version is determined on the first initialize, and the tool must be
//     present in the startup tool list for clients (e.g. Copilot Studio) that import tools
//     once and do not re-fetch after registration changes.
//   - useAiTextEmbed: evaluated at call time, same lazy-registration rationale as
//     useSearchClause — true → embed via ai.text.embed (Neo4j ≥ 2025.11); false → embed via
//     the genai.vector.encode fallback (5.x, 2025.01–2025.10, incl. current Aura 5.x).
func SearchHandler(deps *tools.ToolDependencies, emb config.EmbeddingConfig, useSearchClause func() bool, useAiTextEmbed func() bool) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleVectorSearch(ctx, request, deps, emb, useSearchClause(), useAiTextEmbed())
	}
}

func handleVectorSearch(
	ctx context.Context,
	request mcp.CallToolRequest,
	deps *tools.ToolDependencies,
	emb config.EmbeddingConfig,
	useSearchClause bool,
	useAiTextEmbed bool,
) (*mcp.CallToolResult, error) {
	// Guard: DB service must be initialised.
	if deps.DBService == nil {
		msg := "database service is not initialized"
		slog.Error(msg)
		return mcp.NewToolResultError(msg), nil
	}

	// Guard: embedding must be configured.
	if emb.Provider == "" {
		msg := "vector-search is not available: embedding provider is not configured (set NEO4J_EMBEDDING_PROVIDER and related env vars)"
		slog.Error(msg)
		return mcp.NewToolResultError(msg), nil
	}

	// Bind arguments.
	var args SearchInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding vector-search arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate query.
	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required and cannot be empty"), nil
	}

	// Apply TopK default and clamp.
	topK := args.TopK
	if topK <= 0 {
		topK = defaultTopK
	}
	if topK > maxTopK {
		topK = maxTopK
	}

	// Validate filters.
	for i, f := range args.Filters {
		if !safePropertyName.MatchString(f.Property) {
			return mcp.NewToolResultError(fmt.Sprintf(
				"filter[%d]: property '%s' is not a valid identifier; must match ^[A-Za-z_][A-Za-z0-9_]*$",
				i, f.Property,
			)), nil
		}
		if !allowedOperators[normalizeOperator(f.Operator)] {
			return mcp.NewToolResultError(fmt.Sprintf(
				"filter[%d]: operator '%s' is not allowed; permitted: =, <>, <, <=, >, >=, IN, CONTAINS, STARTS WITH, ENDS WITH",
				i, f.Operator,
			)), nil
		}
	}

	// Resolve the vector index.
	idx, err := ResolveIndex(ctx, deps.DBService, args.IndexName)
	if err != nil {
		slog.Error("vector-search: failed to resolve index", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build embedding configuration map (never echoed to client). ai.text.embed and
	// genai.vector.encode use different provider literals and config key shapes, so the
	// builder is selected based on the same version gate as the embed function itself.
	var provider string
	var embedConfig map[string]any
	if useAiTextEmbed {
		provider, embedConfig, err = BuildEmbedConfig(emb)
	} else {
		provider, embedConfig, err = BuildGenaiVectorEncodeConfig(emb)
	}
	if err != nil {
		slog.Error("vector-search: embedding config error", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build filter clause.
	filterClause, filterParams, err := BuildFilterClause(args.Filters)
	if err != nil {
		slog.Error("vector-search: filter clause error", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Compute fetchK: over-fetch when post-filtering so we have enough candidates.
	fetchK := topK
	if len(args.Filters) > 0 {
		fetchK = topK * overFetchFactor
		if fetchK > maxFetchK {
			fetchK = maxFetchK
		}
	}

	// Build the Cypher query.
	cypher, err := BuildVectorQuery(idx, useSearchClause, useAiTextEmbed, filterClause)
	if err != nil {
		slog.Error("vector-search: query build error", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Assemble parameters. The embedConfig map is a bound parameter — never
	// interpolated into query text — keeping the API key secure.
	params := map[string]any{
		"query":       args.Query,
		"provider":    provider,
		"embedConfig": embedConfig,
		"topK":        topK,
		"fetchK":      fetchK,
	}
	// queryNodes path needs the index name as a parameter (SEARCH path uses a literal).
	if !useSearchClause {
		params["indexName"] = idx.Name
	}
	// Merge filter value parameters.
	for k, v := range filterParams {
		params[k] = v
	}

	slog.Info("executing vector-search query",
		"index", idx.Name,
		"topK", topK,
		"fetchK", fetchK,
		"filters", len(args.Filters),
		"useSearchClause", useSearchClause,
		"useAiTextEmbed", useAiTextEmbed,
	)

	// Execute — errors are sanitised before returning to the client (§3/§11).
	records, err := deps.DBService.ExecuteReadQuery(ctx, cypher, params)
	if err != nil {
		// Log the full error server-side but never return it to the client (it may
		// contain the $embedConfig value depending on driver error formatting).
		slog.Error("vector-search: query execution failed", "error", err)
		return mcp.NewToolResultError(
			"vector search failed; ensure the GenAI plugin is installed and " +
				"the embedding provider credentials are valid. Check server logs for details.",
		), nil
	}

	// Format results.
	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("vector-search: failed to format results", "error", err)
		return mcp.NewToolResultError("vector search failed while formatting results; check server logs for details"), nil
	}

	return mcp.NewToolResultText(response), nil
}

// normalizeOperator normalises an operator string for lookup in allowedOperators.
func normalizeOperator(op string) string {
	upper := strings.ToUpper(strings.TrimSpace(op))
	return strings.Join(strings.Fields(upper), " ")
}
