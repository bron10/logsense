package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONStrategy detects and parses JSON log lines.
type JSONStrategy struct{}

// Parse checks if the line starts with '{' and attempts JSON unmarshalling.
func (s *JSONStrategy) Parse(line string) map[string]string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") {
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil
	}

	result := make(map[string]string, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			result[k] = val
		case float64:
			if val == float64(int64(val)) {
				result[k] = fmt.Sprintf("%d", int64(val))
			} else {
				result[k] = fmt.Sprintf("%g", val)
			}
		case bool:
			result[k] = fmt.Sprintf("%t", val)
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}
