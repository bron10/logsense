package parser

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	levelRe     = regexp.MustCompile(`^\[(\w+)\]`)
	timestampRe = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)`)
	kvRe        = regexp.MustCompile(`(\w+)=("(?:[^"\\]|\\.)*"|\S+)`)
	unitSuffixRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)(ms|s|%)$`)
)

// RegexStrategy parses log lines with [LEVEL] prefixes and key=value pairs.
type RegexStrategy struct{}

// Parse extracts level, timestamp, and key=value pairs from a log line.
func (s *RegexStrategy) Parse(line string) map[string]string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	result := make(map[string]string)

	// Extract log level
	if m := levelRe.FindStringSubmatch(trimmed); m != nil {
		result["level"] = strings.ToLower(m[1])
	}

	// Extract timestamp
	if m := timestampRe.FindStringSubmatch(trimmed); m != nil {
		result["timestamp"] = m[1]
	}

	// Extract key=value pairs
	matches := kvRe.FindAllStringSubmatch(trimmed, -1)
	for _, m := range matches {
		key := m[1]
		value := m[2]

		// Strip surrounding quotes
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		// Try to strip unit suffixes and convert to numeric
		if um := unitSuffixRe.FindStringSubmatch(value); um != nil {
			if _, err := strconv.ParseFloat(um[1], 64); err == nil {
				result[key] = um[1]
				result[key+"_unit"] = um[2]
				continue
			}
		}

		result[key] = value
	}

	// Only return if we extracted something meaningful
	if len(result) == 0 {
		return nil
	}
	return result
}
