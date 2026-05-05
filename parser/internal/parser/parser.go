package parser

// ParseResult is the structured output of parsing
type ParseResult struct {
	Fields   map[string]string
	Strategy string
	Status   string
}

// Strategy defines the interface for parsing strategies
type Strategy interface {
	Parse(line string) map[string]string
}

// Parser chains multiple strategies
type Parser struct {
	strategies []Strategy
}

// New creates parser with JSON first, then Regex
func New() *Parser {
	return &Parser{
		strategies: []Strategy{
			&JSONStrategy{},
			&RegexStrategy{},
		},
	}
}

// Parse runs line through strategies and returns ParseResult
func (p *Parser) Parse(line string) ParseResult {
	for _, s := range p.strategies {
		if fields := s.Parse(line); fields != nil {

			strategy := "unknown"

			switch s.(type) {
			case *JSONStrategy:
				strategy = "json"
			case *RegexStrategy:
				strategy = "regex"
			}

			return ParseResult{
				Fields:   fields,
				Strategy: strategy,
				Status:   "parsed",
			}
		}
	}

	// fallback case
	return ParseResult{
		Fields: map[string]string{
			"raw": line,
		},
		Strategy: "raw",
		Status:   "fallback",
	}
}