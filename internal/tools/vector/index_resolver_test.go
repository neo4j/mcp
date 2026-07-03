// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/tools/vector"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

// makeIndexRecord builds a *neo4j.Record that mimics a row from SHOW VECTOR INDEXES.
func makeIndexRecord(name, entityType, label, embProp string, dims any) *neo4j.Record {
	optMap := map[string]any{}
	if dims != nil {
		optMap["indexConfig"] = map[string]any{
			"vector.dimensions": dims,
		}
	}
	return &neo4j.Record{
		Keys: []string{"name", "entityType", "labelsOrTypes", "properties", "options"},
		Values: []any{
			name,
			entityType,
			[]any{label},
			[]any{embProp},
			optMap,
		},
	}
}

func TestResolveIndex_SingleAutoSelect(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{
			makeIndexRecord("moviePlots", "NODE", "Movie", "embedding", int64(1536)),
		}, nil)

	idx, err := vector.ResolveIndex(context.Background(), mockDB, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Name != "moviePlots" {
		t.Errorf("name: want moviePlots, got %s", idx.Name)
	}
	if idx.Label != "Movie" {
		t.Errorf("label: want Movie, got %s", idx.Label)
	}
	if idx.EmbeddingProperty != "embedding" {
		t.Errorf("embProp: want embedding, got %s", idx.EmbeddingProperty)
	}
	if idx.EntityType != "NODE" {
		t.Errorf("entityType: want NODE, got %s", idx.EntityType)
	}
	if idx.Dimensions != 1536 {
		t.Errorf("dimensions: want 1536, got %d", idx.Dimensions)
	}
}

func TestResolveIndex_NamedIndex(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{
			makeIndexRecord("moviePlots", "NODE", "Movie", "embedding", int64(1536)),
			makeIndexRecord("actorBios", "NODE", "Actor", "bio_emb", nil),
		}, nil)

	idx, err := vector.ResolveIndex(context.Background(), mockDB, "actorBios")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Name != "actorBios" {
		t.Errorf("name: want actorBios, got %s", idx.Name)
	}
	if idx.Dimensions != 0 {
		t.Errorf("dimensions: want 0 when absent, got %d", idx.Dimensions)
	}
}

func TestResolveIndex_NotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{
			makeIndexRecord("moviePlots", "NODE", "Movie", "embedding", nil),
		}, nil)

	_, err := vector.ResolveIndex(context.Background(), mockDB, "nonExistentIndex")
	if err == nil {
		t.Fatal("expected error for not-found index")
	}
	if !strings.Contains(err.Error(), "nonExistentIndex") {
		t.Errorf("error should mention the missing index name: %v", err)
	}
	if !strings.Contains(err.Error(), "moviePlots") {
		t.Errorf("error should list available indexes: %v", err)
	}
}

func TestResolveIndex_Ambiguous(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{
			makeIndexRecord("moviePlots", "NODE", "Movie", "embedding", nil),
			makeIndexRecord("actorBios", "NODE", "Actor", "bio_emb", nil),
		}, nil)

	_, err := vector.ResolveIndex(context.Background(), mockDB, "")
	if err == nil {
		t.Fatal("expected error when multiple indexes and no name given")
	}
	if !strings.Contains(err.Error(), "moviePlots") || !strings.Contains(err.Error(), "actorBios") {
		t.Errorf("error should list available indexes: %v", err)
	}
}

func TestResolveIndex_NoIndexes(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{}, nil)

	_, err := vector.ResolveIndex(context.Background(), mockDB, "")
	if err == nil {
		t.Fatal("expected error when no indexes")
	}
	if !strings.Contains(err.Error(), "no vector index") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestResolveIndex_DBError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(nil, errors.New("connection refused"))

	_, err := vector.ResolveIndex(context.Background(), mockDB, "")
	if err == nil {
		t.Fatal("expected error from DB failure")
	}
}

func TestResolveIndex_DimensionsFloat64(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Some versions may return float64 for numeric config values
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{
			makeIndexRecord("moviePlots", "NODE", "Movie", "embedding", float64(768)),
		}, nil)

	idx, err := vector.ResolveIndex(context.Background(), mockDB, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Dimensions != 768 {
		t.Errorf("dimensions from float64: want 768, got %d", idx.Dimensions)
	}
}

func TestResolveIndex_NotFoundNoAvailable(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Asking for a specific index when none exist
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{}, nil)

	_, err := vector.ResolveIndex(context.Background(), mockDB, "missingIndex")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missingIndex") {
		t.Errorf("error should mention index name: %v", err)
	}
}

func TestResolveIndex_RelationshipEntityType(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return([]*neo4j.Record{
			makeIndexRecord("relIdx", "RELATIONSHIP", "ACTED_IN", "plotEmb", nil),
		}, nil)

	idx, err := vector.ResolveIndex(context.Background(), mockDB, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.EntityType != "RELATIONSHIP" {
		t.Errorf("entityType: want RELATIONSHIP, got %s", idx.EntityType)
	}
	if idx.Label != "ACTED_IN" {
		t.Errorf("label: want ACTED_IN, got %s", idx.Label)
	}
}
