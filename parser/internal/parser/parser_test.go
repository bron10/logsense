package parser

import (
	"testing"
)

func TestJSONStrategy_ValidJSON(t *testing.T) {
	s := &JSONStrategy{}
	line := `{"timestamp":"2024-03-15T10:23:00Z","level":"info","event":"volume_attach","volumeId":"vol-0a1b2c3d4e","size_gb":100}`

	result := s.Parse(line)
	if result == nil {
		t.Fatal("expected non-nil result for valid JSON")
	}
	if result["level"] != "info" {
		t.Errorf("expected level=info, got %q", result["level"])
	}
	if result["event"] != "volume_attach" {
		t.Errorf("expected event=volume_attach, got %q", result["event"])
	}
	if result["size_gb"] != "100" {
		t.Errorf("expected size_gb=100, got %q", result["size_gb"])
	}
}

func TestJSONStrategy_NonJSON(t *testing.T) {
	s := &JSONStrategy{}
	line := `[INFO] 2024-03-15T10:23:01Z userId=1001 action=login`

	result := s.Parse(line)
	if result != nil {
		t.Errorf("expected nil for non-JSON line, got %v", result)
	}
}

func TestJSONStrategy_InvalidJSON(t *testing.T) {
	s := &JSONStrategy{}
	line := `{not valid json}`

	result := s.Parse(line)
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

func TestRegexStrategy_KVLine(t *testing.T) {
	s := &RegexStrategy{}
	line := `[INFO] 2024-03-15T10:23:01Z userId=1001 requestId=req-a1b2c3 action=login latency=45ms status=200`

	result := s.Parse(line)
	if result == nil {
		t.Fatal("expected non-nil result for KV line")
	}
	if result["level"] != "info" {
		t.Errorf("expected level=info, got %q", result["level"])
	}
	if result["timestamp"] != "2024-03-15T10:23:01Z" {
		t.Errorf("expected timestamp, got %q", result["timestamp"])
	}
	if result["userId"] != "1001" {
		t.Errorf("expected userId=1001, got %q", result["userId"])
	}
	if result["action"] != "login" {
		t.Errorf("expected action=login, got %q", result["action"])
	}
	if result["latency"] != "45" {
		t.Errorf("expected latency=45 (stripped ms), got %q", result["latency"])
	}
	if result["latency_unit"] != "ms" {
		t.Errorf("expected latency_unit=ms, got %q", result["latency_unit"])
	}
}

func TestRegexStrategy_QuotedValues(t *testing.T) {
	s := &RegexStrategy{}
	line := `[ERROR] 2024-03-15T10:23:06Z userId=1004 action=checkout error="payment gateway timeout"`

	result := s.Parse(line)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["error"] != "payment gateway timeout" {
		t.Errorf("expected unquoted error value, got %q", result["error"])
	}
}

func TestRegexStrategy_EmptyLine(t *testing.T) {
	s := &RegexStrategy{}
	result := s.Parse("")
	if result != nil {
		t.Errorf("expected nil for empty line, got %v", result)
	}
}

func TestRegexStrategy_PercentUnit(t *testing.T) {
	s := &RegexStrategy{}
	line := `[WARN] 2024-03-15T10:23:30Z component=diskUsage usage=82% threshold=80%`

	result := s.Parse(line)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["usage"] != "82" {
		t.Errorf("expected usage=82, got %q", result["usage"])
	}
	if result["usage_unit"] != "%" {
		t.Errorf("expected usage_unit=%%, got %q", result["usage_unit"])
	}
}

func TestParser_JSONPreference(t *testing.T) {
	p := New()
	line := `{"timestamp":"2024-03-15T10:23:00Z","level":"info","event":"test"}`

	result := p.Parse(line)
	if result["event"] != "test" {
		t.Errorf("expected JSON strategy to handle JSON line, got %v", result)
	}
}

func TestParser_FallbackToRegex(t *testing.T) {
	p := New()
	line := `[WARN] 2024-03-15T10:23:03Z userId=1001 action=test`

	result := p.Parse(line)
	if result["level"] != "warn" {
		t.Errorf("expected regex strategy to handle KV line, got %v", result)
	}
}

func TestParser_RawFallback(t *testing.T) {
	p := New()
	line := `some random text that doesn't match anything`

	result := p.Parse(line)
	if result["raw"] == "" {
		t.Errorf("expected raw fallback, got %v", result)
	}
}
