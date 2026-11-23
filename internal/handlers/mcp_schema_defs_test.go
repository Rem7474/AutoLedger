package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

func TestMCPSharedDefsFactorRepeatedObjectsAndStillValidateAnswers(t *testing.T) {
	amount := func(desc string) map[string]any {
		return map[string]any{
			"type": "object", "description": desc, "additionalProperties": false, "required": []any{"amount", "currency"},
			"properties": map[string]any{
				"amount":   map[string]any{"type": "string", "description": "decimal string, e.g. 12.34, never rounded"},
				"currency": map[string]any{"type": "string", "description": "ISO 4217 code of the vehicle"},
			},
		}
	}
	schema := map[string]any{
		"type": "object", "additionalProperties": false, "required": []any{"total", "lines"},
		"properties": map[string]any{
			"total": amount("the total"),
			"lines": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "properties": map[string]any{"cost": amount("what it cost"), "refund": amount("what came back")},
			}},
		},
	}
	factored := factorSchema(schema)
	if factored == nil {
		t.Fatal("a repeated amount must be factored")
	}
	raw, _ := json.Marshal(factored)
	if strings.Count(string(raw), "never rounded") != 1 {
		t.Fatalf("the amount is described once: %s", raw)
	}
	if !strings.Contains(string(raw), `"description":"what came back"`) || !strings.Contains(string(raw), `"description":"the total"`) {
		t.Fatalf("a description written on a use stays on it: %s", raw)
	}
	if len(raw) >= len(mustJSON(t, schema)) {
		t.Fatalf("the factored schema must be smaller: %d >= %d", len(raw), len(mustJSON(t, schema)))
	}

	var s jsonschema.Schema
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.Resolve(nil)
	if err != nil {
		t.Fatalf("the factored schema must resolve: %v", err)
	}
	good := map[string]any{"total": map[string]any{"amount": "1.00", "currency": "EUR"}, "lines": []any{map[string]any{"cost": map[string]any{"amount": "2.00", "currency": "EUR"}}}}
	if err := resolved.Validate(good); err != nil {
		t.Fatalf("a good answer validates: %v", err)
	}
	bad := map[string]any{"total": map[string]any{"amount": 1, "currency": "EUR"}, "lines": []any{}}
	if err := resolved.Validate(bad); err == nil {
		t.Fatal("a wrong amount must not validate")
	}

	if factorSchema(map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "string"}}}) != nil {
		t.Fatal("a schema with nothing repeated is left alone")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
