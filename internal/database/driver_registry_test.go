// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database_test

import (
	"context"
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerRequestDriverRegistry_GetDriver(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr string
	}{
		{
			name: "bolt URI returns driver",
			uri:  "bolt://localhost:7687",
		},
		{
			name: "neo4j URI returns driver",
			uri:  "neo4j://localhost:7687",
		},
		{
			name:    "empty URI returns error",
			uri:     "",
			wantErr: "failed to create Neo4j driver",
		},
		{
			name:    "http scheme returns error",
			uri:     "http://localhost:7687",
			wantErr: "failed to create Neo4j driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &database.PerRequestDriverRegistry{}
			driver, err := registry.GetDriver(tt.uri)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Nil(t, driver)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, driver)
			_ = driver.Close(context.Background())
		})
	}
}
