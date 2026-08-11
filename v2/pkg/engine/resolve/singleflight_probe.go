package resolve

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

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

func singleFlightProbeLog(event string, fields map[string]any) {
	if !singleFlightProbeEnabled() {
		return
	}
	fields["event"] = event
	fields["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	payload, err := json.Marshal(fields)
	if err != nil {
		return
	}
	log.Printf("[singleflight-probe] %s", payload)
}
