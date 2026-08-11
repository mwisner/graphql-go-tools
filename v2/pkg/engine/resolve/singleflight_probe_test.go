package resolve

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/cespare/xxhash/v2"
)

func TestSingleFlightProbeJSONFieldHash(t *testing.T) {
	input := []byte(`{"header":{"X-Test":"header-value"},"body":{"query":"query Test { viewer { id } }","variables":{"id":"variable-value"},"extensions":{"requestId":"extension-value"}}}`)

	tests := []struct {
		name    string
		path    []string
		value   string
		present bool
	}{
		{name: "query", path: []string{"body", "query"}, value: `query Test { viewer { id } }`, present: true},
		{name: "variables", path: []string{"body", "variables"}, value: `{"id":"variable-value"}`, present: true},
		{name: "extensions", path: []string{"body", "extensions"}, value: `{"requestId":"extension-value"}`, present: true},
		{name: "embedded header", path: []string{"header"}, value: `{"X-Test":"header-value"}`, present: true},
		{name: "missing", path: []string{"body", "missing"}, present: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, present := singleFlightProbeJSONFieldHash(input, tt.path...)
			if present != tt.present {
				t.Fatalf("present = %v, want %v", present, tt.present)
			}
			if !tt.present {
				if hash != "" {
					t.Fatalf("hash = %q, want empty", hash)
				}
				return
			}
			want := fmt.Sprintf("%016x", xxhash.Sum64String(tt.value))
			if hash != want {
				t.Fatalf("hash = %q, want %q", hash, want)
			}
		})
	}
}

func TestSingleFlightProbeJSONFieldSummary(t *testing.T) {
	input := []byte(`{"body":{"extensions":{"requestId":"secret-value","trace":{"sampled":true,"spanId":"span-secret"},"tags":["private-value"]}}}`)

	fields, truncated, present := singleFlightProbeJSONFieldSummary(input, "body", "extensions")
	if !present {
		t.Fatal("present = false, want true")
	}
	if truncated {
		t.Fatal("truncated = true, want false")
	}

	want := []singleFlightProbeJSONField{
		{Path: "requestId", Type: "string", Hash: singleFlightProbeCanonicalJSONHash("secret-value")},
		{Path: "tags", Type: "array", Hash: singleFlightProbeCanonicalJSONHash([]any{"private-value"})},
		{Path: "trace", Type: "object", Hash: singleFlightProbeCanonicalJSONHash(map[string]any{"sampled": true, "spanId": "span-secret"})},
		{Path: "trace.sampled", Type: "boolean", Hash: singleFlightProbeCanonicalJSONHash(true)},
		{Path: "trace.spanId", Type: "string", Hash: singleFlightProbeCanonicalJSONHash("span-secret")},
	}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("fields = %#v, want %#v", fields, want)
	}
}

func TestSingleFlightProbeJSONFieldSummaryCanonicalizesObjectOrder(t *testing.T) {
	first := []byte(`{"body":{"extensions":{"trace":{"spanId":"same","sampled":true}}}}`)
	second := []byte(`{"body":{"extensions":{"trace":{"sampled":true,"spanId":"same"}}}}`)

	firstFields, _, _ := singleFlightProbeJSONFieldSummary(first, "body", "extensions")
	secondFields, _, _ := singleFlightProbeJSONFieldSummary(second, "body", "extensions")
	if !reflect.DeepEqual(firstFields, secondFields) {
		t.Fatalf("field summaries differ for equivalent objects: %#v != %#v", firstFields, secondFields)
	}
}

func TestSingleFlightProbeJSONFieldRawEmbedsOriginalValue(t *testing.T) {
	input := []byte(`{"body":{"extensions":{"requestId":"raw-value","trace":{"spanId":"span-value"}}}}`)

	raw, present := singleFlightProbeJSONFieldRaw(input, "body", "extensions")
	if !present {
		t.Fatal("present = false, want true")
	}
	payload, err := json.Marshal(map[string]any{"extensions_raw": raw})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"extensions_raw":{"requestId":"raw-value","trace":{"spanId":"span-value"}}}`
	if string(payload) != want {
		t.Fatalf("payload = %s, want %s", payload, want)
	}
}
