package resolve

import (
	"fmt"
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
