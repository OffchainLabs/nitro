// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

// ErrStylusCallDepthExceeded is returned by TxProcessor.ExecuteWASM when the
// configured node-level Stylus call-depth cap (ArbNodeConfig.MaxStylusCallDepth)
// is reached. It is distinct from vm.ErrDepth, which represents Ethereum's
// 1024-frame EVM call-stack limit.
var ErrStylusCallDepthExceeded = errors.New("stylus call depth exceeded")

// WarnStylusOpenPagesThreshold is the threshold below which Validate() emits a
// warning. One WASM page is 64 KiB, so 4 pages = 256 KiB. Values below this
// are unusually low and may cause most Stylus transactions to be rejected.
const WarnStylusOpenPagesThreshold uint16 = 4

// WarnStylusCallDepthThreshold is the threshold below which Validate() emits a
// warning for MaxStylusCallDepth. A limit of 1 rejects any nested Stylus call
// and is almost certainly a misconfiguration.
const WarnStylusCallDepthThreshold uint16 = 2

// ArbNodeConfig carries Nitro-defined, node-level configuration through the geth
// state.Database boundary. Geth stores it as `any` (see state.Database.ArbNodeConfig);
// Nitro reads it back via a type assertion at the call site. Add new node-level
// knobs as fields here rather than growing the geth interface.
type ArbNodeConfig struct {
	// MaxOpenPages is the per-transaction limit on open Stylus WASM pages.
	// 0 disables the limit.
	MaxOpenPages uint16

	// MaxStylusCallDepth is the per-transaction limit on the number of Stylus
	// frames simultaneously on the call stack. It counts only Stylus frames;
	// EVM frames between two Stylus frames are not counted and do not reset
	// the counter. 0 disables the limit.
	MaxStylusCallDepth uint16
}

// Validate checks ArbNodeConfig fields and logs warnings if configured limits
// are unusually low.
func (c *ArbNodeConfig) Validate() {
	if c.MaxOpenPages > 0 && c.MaxOpenPages < WarnStylusOpenPagesThreshold {
		log.Warn("max-stylus-open-pages is very low; most Stylus transactions may be rejected",
			"value", c.MaxOpenPages, "threshold", WarnStylusOpenPagesThreshold)
	}
	if c.MaxStylusCallDepth > 0 && c.MaxStylusCallDepth < WarnStylusCallDepthThreshold {
		log.Warn("max-stylus-call-depth is very low; most nested Stylus calls may be rejected",
			"value", c.MaxStylusCallDepth, "threshold", WarnStylusCallDepthThreshold)
	}
}

// GetArbNodeConfig returns the ArbNodeConfig stored on the state database, or
// nil if none is set or the stored value has an unexpected type. The wrong-type
// path is a wiring bug: it logs an error and fails open so the node continues
// to run with pre-feature behavior.
func GetArbNodeConfig(statedb vm.StateDB) *ArbNodeConfig {
	raw := statedb.Database().ArbNodeConfig()
	if raw == nil {
		return nil
	}
	cfg, ok := raw.(*ArbNodeConfig)
	if !ok {
		log.Error("ArbNodeConfig unexpected type; node-level Stylus limits inactive",
			"type", fmt.Sprintf("%T", raw))
		return nil
	}
	return cfg
}
