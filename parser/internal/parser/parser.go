package parser

// Strategy defines the interface for log line parsing strategies.
type Strategy interface {
	// Parse attempts to parse a log line into structured fields.
	// Returns nil if this strategy cannot handle the line.
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
// Returns structured fields from the first strategy that succeeds,
// or a raw fallback map if none match.
func (p *Parser) Parse(line string) map[string]string {
	for _, s := range p.strategies {
		if result := s.Parse(line); result != nil {
			return result
		}
	}
	return map[string]string{"raw": line}
}
