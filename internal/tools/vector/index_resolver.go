// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/mcp/internal/database"
)

const showVectorIndexesQuery = `SHOW VECTOR INDEXES YIELD name, entityType, labelsOrTypes, properties, options RETURN name, entityType, labelsOrTypes, properties, options`

// rawIndexEntry is a helper used only within ResolveIndex.
type rawIndexEntry struct {
	name       string
	entityType string
	label      string
	embProp    string
	dimensions int64
}

// ResolveIndex resolves a vector index by name (or auto-selects if only one exists).
// It queries SHOW VECTOR INDEXES and returns full metadata about the resolved index.
func ResolveIndex(ctx context.Context, db database.Service, indexName string) (*ResolvedIndex, error) {
	records, err := db.ExecuteReadQuery(ctx, showVectorIndexesQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list vector indexes: %w", err)
	}

	var all []rawIndexEntry
	for _, rec := range records {
		nameRaw, ok := rec.Get("name")
		if !ok {
			continue
		}
		name, ok := nameRaw.(string)
		if !ok {
			continue
		}

		entityTypeRaw, ok := rec.Get("entityType")
		if !ok {
			continue
		}
		entityType, ok := entityTypeRaw.(string)
		if !ok {
			continue
		}

		labelsRaw, ok := rec.Get("labelsOrTypes")
		if !ok {
			continue
		}
		labels, ok := labelsRaw.([]any)
		if !ok || len(labels) == 0 {
			continue
		}
		label, ok := labels[0].(string)
		if !ok {
			continue
		}

		propsRaw, ok := rec.Get("properties")
		if !ok {
			continue
		}
		props, ok := propsRaw.([]any)
		if !ok || len(props) == 0 {
			continue
		}
		embProp, ok := props[0].(string)
		if !ok {
			continue
		}

		var dims int64
		optionsRaw, ok := rec.Get("options")
		if ok && optionsRaw != nil {
			if optMap, ok := optionsRaw.(map[string]any); ok {
				if idxCfgRaw, ok := optMap["indexConfig"]; ok {
					if idxCfg, ok := idxCfgRaw.(map[string]any); ok {
						if dimRaw, ok := idxCfg["vector.dimensions"]; ok {
							switch v := dimRaw.(type) {
							case int64:
								dims = v
							case float64:
								dims = int64(v)
							}
						}
					}
				}
			}
		}

		all = append(all, rawIndexEntry{
			name:       name,
			entityType: entityType,
			label:      label,
			embProp:    embProp,
			dimensions: dims,
		})
	}

	var selected *rawIndexEntry

	if indexName != "" {
		for i := range all {
			if all[i].name == indexName {
				selected = &all[i]
				break
			}
		}
		if selected == nil {
			available := collectNames(all)
			if len(available) == 0 {
				return nil, fmt.Errorf("vector index '%s' not found; no vector indexes exist in the database", indexName)
			}
			return nil, fmt.Errorf("vector index '%s' not found; available vector indexes: %s", indexName, strings.Join(available, ", "))
		}
	} else {
		switch len(all) {
		case 0:
			return nil, fmt.Errorf("no vector index found in the database")
		case 1:
			selected = &all[0]
		default:
			return nil, fmt.Errorf("multiple vector indexes found (%s); specify indexName to disambiguate", strings.Join(collectNames(all), ", "))
		}
	}

	return &ResolvedIndex{
		Name:              selected.name,
		EntityType:        selected.entityType,
		Label:             selected.label,
		EmbeddingProperty: selected.embProp,
		Dimensions:        selected.dimensions,
	}, nil
}

// escapeIdentifier validates that the identifier is non-empty and returns it
// wrapped in backticks, with any internal backtick characters doubled.
// This is safe for literal interpolation of index names and labels in Cypher.
func escapeIdentifier(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("identifier must not be empty")
	}
	escaped := strings.ReplaceAll(s, "`", "``")
	return "`" + escaped + "`", nil
}

func collectNames(entries []rawIndexEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}
