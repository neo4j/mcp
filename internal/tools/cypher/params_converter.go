package cypher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// BindArgs unmarshals the tool call arguments and converts numbers.
func BindArgs(request any, target any) error {
	// Marshal the request to JSON.
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request to JSON: %w", err)
	}

	// Unmarshal into the final target struct, using json.Number to preserve number types.
	// This ensures that numbers are initially parsed as json.Number.
	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to unmarshal request to target: %w", err)
	}

	// Now, iterate through the target's Params field and convert numbers.
	// This assumes 'target' is a struct with a 'Params' field of type map[string]any.
	// We need to use reflection to access and modify the Params field.
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		paramsField := val.FieldByName("Params")
		if paramsField.IsValid() && paramsField.CanSet() && paramsField.Kind() == reflect.Map {
			// Convert the map to a modifiable map[string]any
			convertedParams := ConvertNumbers(paramsField.Interface()).(map[string]any)
			paramsField.Set(reflect.ValueOf(convertedParams))
		}
	}

	return nil
}

// ConvertNumbers recursively traverses a map or slice and converts float64 to int where possible.
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
