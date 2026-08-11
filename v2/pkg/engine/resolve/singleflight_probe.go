package resolve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/buger/jsonparser"
	"github.com/cespare/xxhash/v2"
)

const (
	singleFlightProbeEnabledEnv    = "WG_SINGLEFLIGHT_PROBE"
	singleFlightProbeDataSourceEnv = "WG_SINGLEFLIGHT_PROBE_DATASOURCE"
)

var singleFlightProbeActiveSubscriptionUpdates atomic.Int64

const (
	singleFlightProbeMaxJSONFields = 64
	singleFlightProbeMaxJSONDepth  = 6
)

type singleFlightProbeJSONField struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Hash string `json:"hash"`
}

func singleFlightProbeEnabled() bool {
	enabled := strings.TrimSpace(os.Getenv(singleFlightProbeEnabledEnv))
	return strings.EqualFold(enabled, "true") || enabled == "1"
}

func singleFlightProbeMatchesDataSource(dataSourceName string) bool {
	filter := strings.TrimSpace(os.Getenv(singleFlightProbeDataSourceEnv))
	return filter == "" || strings.EqualFold(filter, dataSourceName)
}

func singleFlightProbeHash(input []byte) uint64 {
	return xxhash.Sum64(input)
}

func singleFlightProbeJSONFieldHash(input []byte, path ...string) (hash string, present bool) {
	value, _, _, err := jsonparser.Get(input, path...)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(value)), true
}

func singleFlightProbeCanonicalJSONHash(value any) string {
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(canonical))
}

func singleFlightProbeJSONType(value any) string {
	switch value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// singleFlightProbeJSONFieldSummary exposes only field paths, JSON types, and
// hashes. It intentionally never returns raw values. Objects are traversed in
// sorted-key order so equivalent objects produce the same summary regardless
// of their rendered key order. Arrays are hashed as a unit to keep log volume
// and field-path cardinality bounded.
func singleFlightProbeJSONFieldSummary(input []byte, path ...string) (fields []singleFlightProbeJSONField, truncated, present bool) {
	value, _, _, err := jsonparser.Get(input, path...)
	if err != nil {
		return nil, false, false
	}

	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false, true
	}

	var walkObject func(object map[string]any, prefix string, depth int)
	walkObject = func(object map[string]any, prefix string, depth int) {
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			if len(fields) >= singleFlightProbeMaxJSONFields {
				truncated = true
				return
			}
			fieldPath := key
			if prefix != "" {
				fieldPath = prefix + "." + key
			}
			fieldValue := object[key]
			fields = append(fields, singleFlightProbeJSONField{
				Path: fieldPath,
				Type: singleFlightProbeJSONType(fieldValue),
				Hash: singleFlightProbeCanonicalJSONHash(fieldValue),
			})

			child, ok := fieldValue.(map[string]any)
			if !ok {
				continue
			}
			if depth >= singleFlightProbeMaxJSONDepth {
				truncated = true
				continue
			}
			walkObject(child, fieldPath, depth+1)
			if truncated && len(fields) >= singleFlightProbeMaxJSONFields {
				return
			}
		}
	}

	object, ok := decoded.(map[string]any)
	if !ok {
		return []singleFlightProbeJSONField{{
			Path: "$",
			Type: singleFlightProbeJSONType(decoded),
			Hash: singleFlightProbeCanonicalJSONHash(decoded),
		}}, false, true
	}
	walkObject(object, "", 0)
	return fields, truncated, true
}

func singleFlightProbeLog(event string, fields map[string]any) {
	if !singleFlightProbeEnabled() {
		return
	}
	fields["component"] = "singleflight_probe"
	fields["event"] = event
	fields["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	payload, err := json.Marshal(fields)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, string(payload))
}
