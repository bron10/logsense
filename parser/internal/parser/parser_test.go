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
	line := `[INFO] something`

	result := s.Parse(line)
	if result != nil {
		t.Errorf("expected nil for non-JSON line")
	}
}

func TestRegexStrategy_KVLine(t *testing.T) {
	s := &RegexStrategy{}
	line := `[INFO] 2024-03-15T10:23:01Z userId=1001 action=login latency=45ms`

	result := s.Parse(line)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result["level"] != "info" {
		t.Errorf("expected level=info, got %q", result["level"])
	}
	if result["userId"] != "1001" {
		t.Errorf("expected userId=1001, got %q", result["userId"])
	}
	if result["latency"] != "45" {
		t.Errorf("expected latency=45, got %q", result["latency"])
	}
	if result["latency_unit"] != "ms" {
		t.Errorf("expected latency_unit=ms, got %q", result["latency_unit"])
	}
}

func TestParser_JSONPreference(t *testing.T) {
	p := New()
	line := `{"level":"info","event":"test"}`

	result := p.Parse(line)

	if result.Fields["event"] != "test" {
		t.Errorf("expected event=test, got %v", result.Fields)
	}
}

func TestParser_Fallback(t *testing.T) {
	p := New()
	line := `random text`

	result := p.Parse(line)

	if result.Fields["raw"] == "" {
		t.Errorf("expected raw fallback, got %v", result.Fields)
	}
}

//
// ===== NEW METADATA TESTS =====
//

func TestParser_Metadata_JSON(t *testing.T) {
	p := New()
	line := `{"level":"info","event":"test"}`

	result := p.Parse(line)

	if result.Strategy != "json" {
		t.Errorf("expected strategy=json, got %s", result.Strategy)
	}
	if result.Status != "parsed" {
		t.Errorf("expected status=parsed, got %s", result.Status)
	}
}

func TestParser_Metadata_Regex(t *testing.T) {
	p := New()
	line := `[INFO] 2024-03-15 userId=1001 action=login`

	result := p.Parse(line)

	if result.Strategy != "regex" {
		t.Errorf("expected strategy=regex, got %s", result.Strategy)
	}
	if result.Status != "parsed" {
		t.Errorf("expected status=parsed, got %s", result.Status)
	}
}

func TestParser_Metadata_Fallback(t *testing.T) {
	p := New()
	line := `random text`

	result := p.Parse(line)

	if result.Strategy != "raw" {
		t.Errorf("expected strategy=raw, got %s", result.Strategy)
	}
	if result.Status != "fallback" {
		t.Errorf("expected status=fallback, got %s", result.Status)
	}
}
