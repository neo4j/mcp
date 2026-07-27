// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector

import (
	"fmt"
	"strconv"

	"github.com/neo4j/mcp/internal/config"
)

// BuildEmbedConfig builds the configuration map for the ai.text.embed Cypher function.
// It returns the provider string, the configuration map, and any validation error.
// The configuration map is passed as a bound Cypher parameter ($embedConfig) and
// never interpolated into query text, keeping the API key safe.
func BuildEmbedConfig(emb config.EmbeddingConfig) (provider string, cfg map[string]any, err error) {
	switch emb.Provider {
	case config.EmbeddingProviderOpenAI:
		if emb.APIKey == "" {
			return "", nil, fmt.Errorf("embedding API key is required for provider '%s'", emb.Provider)
		}
		if emb.Model == "" {
			return "", nil, fmt.Errorf("embedding model is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"token": emb.APIKey,
			"model": emb.Model,
		}
		if emb.Dimensions != "" {
			dims, err := parseDimensions(emb.Dimensions)
			if err != nil {
				return "", nil, err
			}
			m["vendorOptions"] = map[string]any{"dimensions": dims}
		}
		return emb.Provider, m, nil

	case config.EmbeddingProviderAzureOpenAI:
		if emb.APIKey == "" {
			return "", nil, fmt.Errorf("embedding API key is required for provider '%s'", emb.Provider)
		}
		if emb.Model == "" {
			return "", nil, fmt.Errorf("embedding model is required for provider '%s'", emb.Provider)
		}
		if emb.AzureResource == "" {
			return "", nil, fmt.Errorf("resource name is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"token":    emb.APIKey,
			"resource": emb.AzureResource,
			"model":    emb.Model,
		}
		if emb.Dimensions != "" {
			dims, err := parseDimensions(emb.Dimensions)
			if err != nil {
				return "", nil, err
			}
			m["vendorOptions"] = map[string]any{"dimensions": dims}
		}
		return emb.Provider, m, nil

	case config.EmbeddingProviderVertexAI:
		if emb.APIKey == "" {
			return "", nil, fmt.Errorf("embedding API key (token) is required for provider '%s'", emb.Provider)
		}
		if emb.Model == "" {
			return "", nil, fmt.Errorf("embedding model is required for provider '%s'", emb.Provider)
		}
		if emb.VertexProject == "" {
			return "", nil, fmt.Errorf("project is required for provider '%s'", emb.Provider)
		}
		if emb.VertexRegion == "" {
			return "", nil, fmt.Errorf("region is required for provider '%s'", emb.Provider)
		}
		publisher := emb.VertexPublisher
		if publisher == "" {
			publisher = "google"
		}
		m := map[string]any{
			"token":     emb.APIKey,
			"model":     emb.Model,
			"project":   emb.VertexProject,
			"region":    emb.VertexRegion,
			"publisher": publisher,
		}
		if emb.Dimensions != "" {
			dims, err := parseDimensions(emb.Dimensions)
			if err != nil {
				return "", nil, err
			}
			m["vendorOptions"] = map[string]any{"outputDimensionality": dims}
		}
		return emb.Provider, m, nil

	case config.EmbeddingProviderBedrockTitan:
		if emb.AWSAccessKeyID == "" {
			return "", nil, fmt.Errorf("AWS access key ID is required for provider '%s'", emb.Provider)
		}
		if emb.AWSSecretAccessKey == "" {
			return "", nil, fmt.Errorf("AWS secret access key is required for provider '%s'", emb.Provider)
		}
		if emb.Model == "" {
			return "", nil, fmt.Errorf("embedding model is required for provider '%s'", emb.Provider)
		}
		if emb.AWSRegion == "" {
			return "", nil, fmt.Errorf("AWS region is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"accessKeyId":     emb.AWSAccessKeyID,
			"secretAccessKey": emb.AWSSecretAccessKey,
			"model":           emb.Model,
			"region":          emb.AWSRegion,
		}
		if emb.Dimensions != "" {
			dims, err := parseDimensions(emb.Dimensions)
			if err != nil {
				return "", nil, err
			}
			m["vendorOptions"] = map[string]any{"dimensions": dims}
		}
		return emb.Provider, m, nil

	case "":
		return "", nil, fmt.Errorf("embedding provider is not configured")

	default:
		return "", nil, fmt.Errorf("unknown embedding provider '%s'; must be one of: openai, azure-openai, vertexai, bedrock-titan", emb.Provider)
	}
}

// genaiProviderOpenAI, genaiProviderAzureOpenAI, genaiProviderVertexAI, and
// genaiProviderBedrock are the provider literals expected by the (deprecated but still
// present on Neo4j 5.x / 2025.01–2025.10, incl. current Aura 5.x) genai.vector.encode
// Cypher function. Unlike ai.text.embed, genai.vector.encode uses PascalCase provider
// names rather than the lowercase-hyphen names in config.EmbeddingConfig.Provider.
const (
	genaiProviderOpenAI      = "OpenAI"
	genaiProviderAzureOpenAI = "AzureOpenAI"
	genaiProviderVertexAI    = "VertexAI"
	genaiProviderBedrock     = "Bedrock"
)

// BuildGenaiVectorEncodeConfig builds the configuration map for the genai.vector.encode
// Cypher function — the fallback embedding function for Neo4j versions older than
// 2025.11 (e.g. 5.x, current Aura 5.x), which do not have ai.text.embed. It returns the
// PascalCase provider literal expected by genai.vector.encode, the configuration map,
// and any validation error. As with BuildEmbedConfig, the configuration map is passed as
// a bound Cypher parameter ($embedConfig) and never interpolated into query text, keeping
// the API key safe. Unlike BuildEmbedConfig, dimensions are a flat "dimensions" key
// rather than nested under "vendorOptions".
func BuildGenaiVectorEncodeConfig(emb config.EmbeddingConfig) (provider string, cfg map[string]any, err error) {
	switch emb.Provider {
	case config.EmbeddingProviderOpenAI:
		if emb.APIKey == "" {
			return "", nil, fmt.Errorf("embedding API key is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"token": emb.APIKey,
		}
		if emb.Model != "" {
			m["model"] = emb.Model
		}
		if emb.Dimensions != "" {
			dims, err := parseDimensions(emb.Dimensions)
			if err != nil {
				return "", nil, err
			}
			m["dimensions"] = dims
		}
		return genaiProviderOpenAI, m, nil

	case config.EmbeddingProviderAzureOpenAI:
		if emb.APIKey == "" {
			return "", nil, fmt.Errorf("embedding API key is required for provider '%s'", emb.Provider)
		}
		if emb.AzureResource == "" {
			return "", nil, fmt.Errorf("resource name is required for provider '%s'", emb.Provider)
		}
		if emb.Model == "" {
			return "", nil, fmt.Errorf("embedding model is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"token":      emb.APIKey,
			"resource":   emb.AzureResource,
			"deployment": emb.Model,
		}
		if emb.Dimensions != "" {
			dims, err := parseDimensions(emb.Dimensions)
			if err != nil {
				return "", nil, err
			}
			m["dimensions"] = dims
		}
		return genaiProviderAzureOpenAI, m, nil

	case config.EmbeddingProviderVertexAI:
		if emb.APIKey == "" {
			return "", nil, fmt.Errorf("embedding API key (token) is required for provider '%s'", emb.Provider)
		}
		if emb.VertexProject == "" {
			return "", nil, fmt.Errorf("project is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"token":     emb.APIKey,
			"projectId": emb.VertexProject,
		}
		if emb.Model != "" {
			m["model"] = emb.Model
		}
		if emb.VertexRegion != "" {
			m["region"] = emb.VertexRegion
		}
		return genaiProviderVertexAI, m, nil

	case config.EmbeddingProviderBedrockTitan:
		if emb.AWSAccessKeyID == "" {
			return "", nil, fmt.Errorf("AWS access key ID is required for provider '%s'", emb.Provider)
		}
		if emb.AWSSecretAccessKey == "" {
			return "", nil, fmt.Errorf("AWS secret access key is required for provider '%s'", emb.Provider)
		}
		m := map[string]any{
			"accessKeyId":     emb.AWSAccessKeyID,
			"secretAccessKey": emb.AWSSecretAccessKey,
		}
		if emb.Model != "" {
			m["model"] = emb.Model
		}
		if emb.AWSRegion != "" {
			m["region"] = emb.AWSRegion
		}
		return genaiProviderBedrock, m, nil

	case "":
		return "", nil, fmt.Errorf("embedding provider is not configured")

	default:
		return "", nil, fmt.Errorf("unknown embedding provider '%s'; must be one of: openai, azure-openai, vertexai, bedrock-titan", emb.Provider)
	}
}

// parseDimensions parses the dimensions string to an int64.
func parseDimensions(s string) (int64, error) {
	d, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid embedding dimensions value '%s': must be a positive integer", s)
	}
	return d, nil
}
