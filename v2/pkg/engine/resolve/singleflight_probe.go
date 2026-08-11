package resolve

import (
	"encoding/json"
	"fmt"
	"os"
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
