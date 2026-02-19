//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Params is a map of Cypher query parameters with custom JSON unmarshaling
// that preserves numeric types correctly for Neo4j.
//
// When unmarshaling from JSON:
//   - Whole numbers (e.g., 1, 42, -10) become int64
//   - Numbers with fractional parts (e.g., 1.5, 3.14) become float64
//   - Numbers with decimal notation but no fraction (e.g., 10.0) become float64
//   - Other types (strings, booleans, null) are preserved as-is
type Params map[string]any

func (cp *Params) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	decoder.UseNumber()

	var temp map[string]any
	if err := decoder.Decode(&temp); err != nil {
		return err
	}
	paramConverted, ok := ConvertNumbers(temp).(map[string]any)
	if !ok {
		return fmt.Errorf("error during Unmarshaling of Params")
	}
	*cp = paramConverted
	return nil
}

func ConvertNumbers(input any) any {
	switch v := input.(type) {
	case json.Number:
		// Try to parse as Int64 first
		if i, err := v.Int64(); err == nil {
			return i
		}
		// If it fails (because of decimal point), parse as Float64
		if f, err := v.Float64(); err == nil {
			return f
		}
		return v.String() // Fallback

	case map[string]any:
		for k, val := range v {
			v[k] = ConvertNumbers(val)
		}
		return v

	case []any:
		for i, val := range v {
			v[i] = ConvertNumbers(val)
		}
		return v
	}
	return input
}
