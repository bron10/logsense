package parser

// Strategy defines the interface for log line parsing strategies.
type Strategy interface {
	Parse(line string) map[string]string
}

// ParseResult is the new structured result
type ParseResult struct {
	Fields   map[string]string
	Strategy string
	Status   string
}

// Parser chains multiple strategies
type Parser struct {
	strategies []Strategy
}

// New creates parser
func New() *Parser {
	return &Parser{
		strategies: []Strategy{
			&JSONStrategy{},
			&RegexStrategy{},
		},
	}
}

// Parse returns ParseResult instead of map
func (p *Parser) Parse(line string) ParseResult {
	for _, s := range p.strategies {
		if fields := s.Parse(line); fields != nil {

			strategy := "regex"
			if _, ok := s.(*JSONStrategy); ok {
				strategy = "json"
			}

			return ParseResult{
				Fields:   fields,
				Strategy: strategy,
				Status:   "parsed",
			}
		}
	}

	return ParseResult{
		Fields:   map[string]string{"raw": line},
		Strategy: "raw",
		Status:   "fallback",
	}
}