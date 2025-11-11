package cypher

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ParamsConverter interface for types that have a Params field
type ParamsConverter interface {
	GetParams() map[string]any
	SetParams(map[string]any)
}

// BindArguments is our custom implementation that preserves integer types
// by using json.Number during unmarshaling, then converting to proper types
func BindArguments(request mcp.CallToolRequest, target any) error {
	// Marshal the arguments to JSON
	jsonBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return err
	}

	// Unmarshal with UseNumber() to preserve integers as json.Number
	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	// Convert the Params field using our ConvertNumbers function
	if converter, ok := target.(ParamsConverter); ok {
		params := converter.GetParams()
		if params != nil {
			if convertedParams, ok := ConvertNumbers(params).(map[string]any); ok {
				converter.SetParams(convertedParams)
			} else {
				// Optionally handle the error case, e.g., log or return an error
				return fmt.Errorf("failed to convert params to map[string]any")
			}
		}
	}

	return nil
}

// ConvertNumbers recursively traverses a map or slice and converts json.Number values to proper int64 or float64
func ConvertNumbers(data any) any {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			v[key] = ConvertNumbers(value)
		}
		return v
	case []any:
		for i, value := range v {
			v[i] = ConvertNumbers(value)
		}
		return v
	case json.Number:
		// Attempt to parse as float64 first to handle numbers like 1.0
		if f, err := v.Float64(); err == nil {
			// If the float has no fractional part, convert to int64
			if f == float64(int64(f)) {
				return int64(f)
			}
			return f
		}
		// Fallback to the original string representation if not a valid number
		return v.String()
	default:
		return v
	}
}
