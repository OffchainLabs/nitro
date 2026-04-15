// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"github.com/ethereum/go-ethereum/log"
)

// WarnStylusOpenPagesThreshold is the threshold below which Validate() emits a
// warning. One WASM page is 64 KiB, so 4 pages = 256 KiB. Values below this
// are unusually low and may cause most Stylus transactions to be rejected.
const WarnStylusOpenPagesThreshold uint16 = 4

// ArbNodeConfig carries Nitro-defined, node-level configuration through the geth
// state.Database boundary. Geth stores it as `any` (see state.Database.ArbNodeConfig);
// Nitro reads it back via a type assertion at the call site. Add new node-level
// knobs as fields here rather than growing the geth interface.
type ArbNodeConfig struct {
	// MaxOpenPages is the per-transaction limit on open Stylus WASM pages.
	// 0 disables the limit.
	MaxOpenPages uint16
}

// Validate checks ArbNodeConfig fields and logs a warning if the configured
// limit is unusually low.
func (c *ArbNodeConfig) Validate() {
	if c.MaxOpenPages > 0 && c.MaxOpenPages < WarnStylusOpenPagesThreshold {
		log.Warn("max-stylus-open-pages is very low; most Stylus transactions may be rejected",
			"value", c.MaxOpenPages, "threshold", WarnStylusOpenPagesThreshold)
	}
}
