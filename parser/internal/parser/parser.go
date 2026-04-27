package parser

type ParseResult struct {
	Fields   map[string]string
	Strategy string
	Status   string
}

// Strategy defines the interface for log parsing strategies.
type Strategy interface {
	Parse(line string) map[string]string
}

// Parser chains multiple strategies and returns the first successful parse.
type Parser struct {
	strategies []Strategy
}

// New creates a Parser with the default strategy chain: JSON, then Regex.
func New() *Parser {
	return &Parser{
		strategies: []Strategy{
			&JSONStrategy{},
			&RegexStrategy{},
		},
	}
}

// Parse runs the line through each strategy in order.
func (p *Parser) Parse(line string) ParseResult {
	for _, s := range p.strategies {
		if result := s.Parse(line); result != nil {

			strategy := "unknown"

			switch s.(type) {
			case *JSONStrategy:
				strategy = "json"
			case *RegexStrategy:
				strategy = "regex"
			}

			return ParseResult{
				Fields:   result,
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
