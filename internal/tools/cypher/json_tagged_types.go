package cypher

import (
	"bytes"
	"encoding/json"
)

type CypherParams map[string]any

func (cp *CypherParams) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	decoder.UseNumber()

	var temp map[string]any
	if err := decoder.Decode(&temp); err != nil {
		return err
	}

	*cp = ConvertNumbers(temp).(map[string]any)
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
