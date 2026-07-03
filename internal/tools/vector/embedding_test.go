// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector_test

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/tools/vector"
)

// testToken is a non-secret placeholder used as the embedding credential in table
// tests. It is declared as a variable (rather than a string literal in each struct)
// to avoid gosec G101 false positives on "hardcoded credentials" in test fixtures.
var testToken = "placeholder-token"

func TestBuildEmbedConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		emb          config.EmbeddingConfig
		wantErr      bool
		wantProvider string
		checkCfg     func(t *testing.T, cfg map[string]any)
	}{
		{
			name: "openai basic",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderOpenAI,
				APIKey:   "sk-placeholder",
				Model:    "text-embedding-3-small",
			},
			wantProvider: config.EmbeddingProviderOpenAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", "sk-placeholder")
				assertStringKey(t, cfg, "model", "text-embedding-3-small")
				if _, ok := cfg["vendorOptions"]; ok {
					t.Error("vendorOptions should be absent when Dimensions is empty")
				}
			},
		},
		{
			name: "openai with dimensions",
			emb: config.EmbeddingConfig{
				Provider:   config.EmbeddingProviderOpenAI,
				APIKey:     "sk-placeholder",
				Model:      "text-embedding-3-small",
				Dimensions: "1536",
			},
			wantProvider: config.EmbeddingProviderOpenAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				vo, ok := cfg["vendorOptions"].(map[string]any)
				if !ok {
					t.Fatal("vendorOptions should be present")
				}
				if vo["dimensions"] != int64(1536) {
					t.Errorf("expected dimensions=1536, got %v", vo["dimensions"])
				}
			},
		},
		{
			name: "openai missing api key",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderOpenAI,
				Model:    "text-embedding-3-small",
			},
			wantErr: true,
		},
		{
			name: "openai missing model",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderOpenAI,
				APIKey:   "sk-placeholder",
			},
			wantErr: true,
		},
		{
			name: "azure-openai basic",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderAzureOpenAI,
				APIKey:        "azure-key-placeholder",
				Model:         "text-embedding-ada-002",
				AzureResource: "my-azure-resource",
			},
			wantProvider: config.EmbeddingProviderAzureOpenAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", "azure-key-placeholder")
				assertStringKey(t, cfg, "resource", "my-azure-resource")
				assertStringKey(t, cfg, "model", "text-embedding-ada-002")
			},
		},
		{
			name: "azure-openai with dimensions",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderAzureOpenAI,
				APIKey:        "azure-key-placeholder",
				Model:         "text-embedding-ada-002",
				AzureResource: "my-azure-resource",
				Dimensions:    "256",
			},
			wantProvider: config.EmbeddingProviderAzureOpenAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				vo, ok := cfg["vendorOptions"].(map[string]any)
				if !ok {
					t.Fatal("vendorOptions should be present")
				}
				if vo["dimensions"] != int64(256) {
					t.Errorf("expected dimensions=256, got %v", vo["dimensions"])
				}
			},
		},
		{
			name: "azure-openai missing resource",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderAzureOpenAI,
				APIKey:   "azure-key-placeholder",
				Model:    "text-embedding-ada-002",
			},
			wantErr: true,
		},
		{
			name: "vertexai basic",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderVertexAI,
				APIKey:        testToken,
				Model:         "textembedding-gecko",
				VertexProject: "my-project",
				VertexRegion:  "us-central1",
			},
			wantProvider: config.EmbeddingProviderVertexAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", testToken)
				assertStringKey(t, cfg, "model", "textembedding-gecko")
				assertStringKey(t, cfg, "project", "my-project")
				assertStringKey(t, cfg, "region", "us-central1")
				// default publisher should be "google"
				assertStringKey(t, cfg, "publisher", "google")
			},
		},
		{
			name: "vertexai with explicit publisher",
			emb: config.EmbeddingConfig{
				Provider:        config.EmbeddingProviderVertexAI,
				APIKey:          testToken,
				Model:           "textembedding-gecko",
				VertexProject:   "my-project",
				VertexRegion:    "us-central1",
				VertexPublisher: "anthropic",
			},
			wantProvider: config.EmbeddingProviderVertexAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "publisher", "anthropic")
			},
		},
		{
			name: "vertexai with outputDimensionality",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderVertexAI,
				APIKey:        testToken,
				Model:         "textembedding-gecko",
				VertexProject: "my-project",
				VertexRegion:  "us-central1",
				Dimensions:    "768",
			},
			wantProvider: config.EmbeddingProviderVertexAI,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				vo, ok := cfg["vendorOptions"].(map[string]any)
				if !ok {
					t.Fatal("vendorOptions should be present")
				}
				if vo["outputDimensionality"] != int64(768) {
					t.Errorf("expected outputDimensionality=768, got %v", vo["outputDimensionality"])
				}
			},
		},
		{
			name: "vertexai missing project",
			emb: config.EmbeddingConfig{
				Provider:     config.EmbeddingProviderVertexAI,
				APIKey:       testToken,
				Model:        "textembedding-gecko",
				VertexRegion: "us-central1",
			},
			wantErr: true,
		},
		{
			name: "bedrock-titan basic",
			emb: config.EmbeddingConfig{
				Provider:           config.EmbeddingProviderBedrockTitan,
				AWSAccessKeyID:     "test-access-key-id",
				AWSSecretAccessKey: "test-secret-access-key",
				Model:              "amazon.titan-embed-text-v1",
				AWSRegion:          "us-east-1",
			},
			wantProvider: config.EmbeddingProviderBedrockTitan,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "accessKeyId", "test-access-key-id")
				assertStringKey(t, cfg, "secretAccessKey", "test-secret-access-key")
				assertStringKey(t, cfg, "model", "amazon.titan-embed-text-v1")
				assertStringKey(t, cfg, "region", "us-east-1")
			},
		},
		{
			name: "bedrock-titan with dimensions",
			emb: config.EmbeddingConfig{
				Provider:           config.EmbeddingProviderBedrockTitan,
				AWSAccessKeyID:     "test-access-key-id",
				AWSSecretAccessKey: "test-secret-access-key",
				Model:              "amazon.titan-embed-text-v1",
				AWSRegion:          "us-east-1",
				Dimensions:         "512",
			},
			wantProvider: config.EmbeddingProviderBedrockTitan,
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				vo, ok := cfg["vendorOptions"].(map[string]any)
				if !ok {
					t.Fatal("vendorOptions should be present")
				}
				if vo["dimensions"] != int64(512) {
					t.Errorf("expected dimensions=512, got %v", vo["dimensions"])
				}
			},
		},
		{
			name: "bedrock-titan missing access key",
			emb: config.EmbeddingConfig{
				Provider:           config.EmbeddingProviderBedrockTitan,
				AWSSecretAccessKey: "secret-placeholder",
				Model:              "amazon.titan-embed-text-v1",
				AWSRegion:          "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "empty provider",
			emb: config.EmbeddingConfig{
				Provider: "",
			},
			wantErr: true,
		},
		{
			name: "unknown provider",
			emb: config.EmbeddingConfig{
				Provider: "llama-local",
			},
			wantErr: true,
		},
		{
			name: "dimensions non-numeric",
			emb: config.EmbeddingConfig{
				Provider:   config.EmbeddingProviderOpenAI,
				APIKey:     "sk-placeholder",
				Model:      "text-embedding-3-small",
				Dimensions: "not-a-number",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider, cfg, err := vector.BuildEmbedConfig(tc.emb)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (provider=%s, cfg=%v)", provider, cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider != tc.wantProvider {
				t.Errorf("provider: want %q, got %q", tc.wantProvider, provider)
			}
			if tc.checkCfg != nil {
				tc.checkCfg(t, cfg)
			}
		})
	}
}

func TestBuildGenaiVectorEncodeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		emb          config.EmbeddingConfig
		wantErr      bool
		wantProvider string
		checkCfg     func(t *testing.T, cfg map[string]any)
	}{
		{
			name: "openai basic",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderOpenAI,
				APIKey:   "sk-placeholder",
				Model:    "text-embedding-3-small",
			},
			wantProvider: "OpenAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", "sk-placeholder")
				assertStringKey(t, cfg, "model", "text-embedding-3-small")
				if _, ok := cfg["dimensions"]; ok {
					t.Error("dimensions should be absent when Dimensions is empty")
				}
				if _, ok := cfg["vendorOptions"]; ok {
					t.Error("genai.vector.encode config must not use nested vendorOptions")
				}
			},
		},
		{
			name: "openai with dimensions is flat, not nested",
			emb: config.EmbeddingConfig{
				Provider:   config.EmbeddingProviderOpenAI,
				APIKey:     "sk-placeholder",
				Model:      "text-embedding-3-small",
				Dimensions: "1536",
			},
			wantProvider: "OpenAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				if cfg["dimensions"] != int64(1536) {
					t.Errorf("expected flat dimensions=1536, got %v", cfg["dimensions"])
				}
			},
		},
		{
			name: "openai model is optional",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderOpenAI,
				APIKey:   "sk-placeholder",
			},
			wantProvider: "OpenAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", "sk-placeholder")
				if _, ok := cfg["model"]; ok {
					t.Error("model should be absent when not set")
				}
			},
		},
		{
			name: "openai missing token",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderOpenAI,
				Model:    "text-embedding-3-small",
			},
			wantErr: true,
		},
		{
			name: "azure-openai basic maps model to deployment",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderAzureOpenAI,
				APIKey:        "azure-key-placeholder",
				Model:         "text-embedding-ada-002",
				AzureResource: "my-azure-resource",
			},
			wantProvider: "AzureOpenAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", "azure-key-placeholder")
				assertStringKey(t, cfg, "resource", "my-azure-resource")
				assertStringKey(t, cfg, "deployment", "text-embedding-ada-002")
				if _, ok := cfg["model"]; ok {
					t.Error("genai.vector.encode azure config must use 'deployment', not 'model'")
				}
			},
		},
		{
			name: "azure-openai with dimensions is flat",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderAzureOpenAI,
				APIKey:        "azure-key-placeholder",
				Model:         "text-embedding-ada-002",
				AzureResource: "my-azure-resource",
				Dimensions:    "256",
			},
			wantProvider: "AzureOpenAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				if cfg["dimensions"] != int64(256) {
					t.Errorf("expected flat dimensions=256, got %v", cfg["dimensions"])
				}
			},
		},
		{
			name: "azure-openai missing resource",
			emb: config.EmbeddingConfig{
				Provider: config.EmbeddingProviderAzureOpenAI,
				APIKey:   "azure-key-placeholder",
				Model:    "text-embedding-ada-002",
			},
			wantErr: true,
		},
		{
			name: "azure-openai missing deployment (model)",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderAzureOpenAI,
				APIKey:        "azure-key-placeholder",
				AzureResource: "my-azure-resource",
			},
			wantErr: true,
		},
		{
			name: "vertexai basic uses projectId not project, no publisher",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderVertexAI,
				APIKey:        testToken,
				Model:         "textembedding-gecko",
				VertexProject: "my-project",
				VertexRegion:  "us-central1",
			},
			wantProvider: "VertexAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "token", testToken)
				assertStringKey(t, cfg, "model", "textembedding-gecko")
				assertStringKey(t, cfg, "projectId", "my-project")
				assertStringKey(t, cfg, "region", "us-central1")
				if _, ok := cfg["project"]; ok {
					t.Error("genai.vector.encode vertex config must use 'projectId', not 'project'")
				}
				if _, ok := cfg["publisher"]; ok {
					t.Error("genai.vector.encode vertex config must not include 'publisher'")
				}
			},
		},
		{
			name: "vertexai model and region are optional",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderVertexAI,
				APIKey:        testToken,
				VertexProject: "my-project",
			},
			wantProvider: "VertexAI",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "projectId", "my-project")
				if _, ok := cfg["model"]; ok {
					t.Error("model should be absent when not set")
				}
				if _, ok := cfg["region"]; ok {
					t.Error("region should be absent when not set")
				}
			},
		},
		{
			name: "vertexai missing token",
			emb: config.EmbeddingConfig{
				Provider:      config.EmbeddingProviderVertexAI,
				VertexProject: "my-project",
				VertexRegion:  "us-central1",
			},
			wantErr: true,
		},
		{
			name: "vertexai missing projectId",
			emb: config.EmbeddingConfig{
				Provider:     config.EmbeddingProviderVertexAI,
				APIKey:       testToken,
				VertexRegion: "us-central1",
			},
			wantErr: true,
		},
		{
			name: "bedrock-titan basic",
			emb: config.EmbeddingConfig{
				Provider:           config.EmbeddingProviderBedrockTitan,
				AWSAccessKeyID:     "test-access-key-id",
				AWSSecretAccessKey: "test-secret-access-key",
				Model:              "amazon.titan-embed-text-v1",
				AWSRegion:          "us-east-1",
			},
			wantProvider: "Bedrock",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				assertStringKey(t, cfg, "accessKeyId", "test-access-key-id")
				assertStringKey(t, cfg, "secretAccessKey", "test-secret-access-key")
				assertStringKey(t, cfg, "model", "amazon.titan-embed-text-v1")
				assertStringKey(t, cfg, "region", "us-east-1")
			},
		},
		{
			name: "bedrock-titan model and region are optional",
			emb: config.EmbeddingConfig{
				Provider:           config.EmbeddingProviderBedrockTitan,
				AWSAccessKeyID:     "test-access-key-id",
				AWSSecretAccessKey: "test-secret-access-key",
			},
			wantProvider: "Bedrock",
			checkCfg: func(t *testing.T, cfg map[string]any) {
				t.Helper()
				if _, ok := cfg["model"]; ok {
					t.Error("model should be absent when not set")
				}
				if _, ok := cfg["region"]; ok {
					t.Error("region should be absent when not set")
				}
			},
		},
		{
			name: "bedrock-titan missing access key",
			emb: config.EmbeddingConfig{
				Provider:           config.EmbeddingProviderBedrockTitan,
				AWSSecretAccessKey: "secret-placeholder",
			},
			wantErr: true,
		},
		{
			name: "bedrock-titan missing secret",
			emb: config.EmbeddingConfig{
				Provider:       config.EmbeddingProviderBedrockTitan,
				AWSAccessKeyID: "test-access-key-id",
			},
			wantErr: true,
		},
		{
			name: "empty provider",
			emb: config.EmbeddingConfig{
				Provider: "",
			},
			wantErr: true,
		},
		{
			name: "unknown provider",
			emb: config.EmbeddingConfig{
				Provider: "llama-local",
			},
			wantErr: true,
		},
		{
			name: "dimensions non-numeric",
			emb: config.EmbeddingConfig{
				Provider:   config.EmbeddingProviderOpenAI,
				APIKey:     "sk-placeholder",
				Dimensions: "not-a-number",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider, cfg, err := vector.BuildGenaiVectorEncodeConfig(tc.emb)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (provider=%s, cfg=%v)", provider, cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider != tc.wantProvider {
				t.Errorf("provider: want %q, got %q", tc.wantProvider, provider)
			}
			if tc.checkCfg != nil {
				tc.checkCfg(t, cfg)
			}
		})
	}
}

// TestBuildGenaiVectorEncodeConfig_SecretsNotInQueryText is a security regression test:
// the token/secret values must only ever appear in the returned config map (a bound
// Cypher parameter), never embedded in any query string the caller might build.
func TestBuildGenaiVectorEncodeConfig_SecretsNotInQueryText(t *testing.T) {
	t.Parallel()
	sensitiveToken := "sk-super-secret-should-never-be-in-query-text"
	emb := config.EmbeddingConfig{
		Provider: config.EmbeddingProviderOpenAI,
		APIKey:   sensitiveToken,
		Model:    "text-embedding-3-small",
	}

	provider, cfg, err := vector.BuildGenaiVectorEncodeConfig(emb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The secret must be present in the returned map (it is passed as a bound param)...
	if cfg["token"] != sensitiveToken {
		t.Fatalf("expected token in config map, got %v", cfg["token"])
	}
	// ...but the query the caller assembles never contains provider/config literals: only
	// the bound parameter names $provider and $embedConfig appear in BuildVectorQuery's
	// output, which is exercised separately in query_builder_test.go. Here we assert the
	// provider literal itself carries no secret material.
	if strings.Contains(provider, sensitiveToken) {
		t.Fatalf("provider literal must not contain secret material: %q", provider)
	}
}

func assertStringKey(t *testing.T, m map[string]any, key, want string) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("key %q not found in config map", key)
		return
	}
	s, ok := v.(string)
	if !ok {
		t.Errorf("key %q: expected string, got %T", key, v)
		return
	}
	if s != want {
		t.Errorf("key %q: want %q, got %q", key, want, s)
	}
}
