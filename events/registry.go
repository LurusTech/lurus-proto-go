// Package events indexes every event type registered on the lurus event
// backbone. The map below is the machine-readable twin of the SCHEMA_RULES.md
// 总表: schema_test.go cross-checks it against the compiled descriptors both
// ways, so an event message cannot exist unregistered and a registry entry
// cannot point at a missing message.
package events

import (
	"google.golang.org/protobuf/proto"

	lugov1 "github.com/hanmahong5-arch/lurus-proto-go/events/lugo/v1"
	syncv1 "github.com/hanmahong5-arch/lurus-proto-go/events/sync/v1"
	tallyv1 "github.com/hanmahong5-arch/lurus-proto-go/events/tally/v1"
)

// Registry maps each event_type NSID to a constructor for its current payload
// message. Adding an event type means adding a row here, in SCHEMA_RULES.md,
// and the message itself — schema_test.go fails on any partial addition.
var Registry = map[string]func() proto.Message{
	"cn.lurus.tally.stock.received":    func() proto.Message { return &tallyv1.StockReceived{} },
	"cn.lurus.tally.stock.issued":      func() proto.Message { return &tallyv1.StockIssued{} },
	"cn.lurus.tally.stock.transferred": func() proto.Message { return &tallyv1.StockTransferred{} },
	"cn.lurus.tally.stock.counted":     func() proto.Message { return &tallyv1.StockCounted{} },
	"cn.lurus.tally.stock.written_off": func() proto.Message { return &tallyv1.StockWrittenOff{} },

	"cn.lurus.lugo.billing.charged":        func() proto.Message { return &lugov1.BillingCharged{} },
	"cn.lurus.lugo.billing.refunded":       func() proto.Message { return &lugov1.BillingRefunded{} },
	"cn.lurus.lugo.billing.quota_deducted": func() proto.Message { return &lugov1.QuotaDeducted{} },

	"cn.lurus.sync.device_registered": func() proto.Message { return &syncv1.DeviceRegistered{} },
	"cn.lurus.sync.cursor_advanced":   func() proto.Message { return &syncv1.CursorAdvanced{} },
}

// New returns an empty payload message for eventType, or ok=false when the
// type is not registered — the machine-checkable definition of "unknown event
// type" for projectors and audit tooling.
func New(eventType string) (proto.Message, bool) {
	c, ok := Registry[eventType]
	if !ok {
		return nil, false
	}
	return c(), true
}
