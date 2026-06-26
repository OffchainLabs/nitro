// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package logfmt

import (
	"github.com/ethereum/go-ethereum/log"
)

func badFormatVerbs() {
	log.Warn("failed init redis for %v: %w", "mr", "err") // want `log message contains printf-style format verb`
	log.Error("count is %d", 5)                           // want `log message contains printf-style format verb`
	log.Info("hash: %s", "abc")                           // want `log message contains printf-style format verb`
	log.Debug("value is %+v", "x")                        // want `log message contains printf-style format verb`
	log.Trace("hex: %02x", 255)                           // want `log message contains printf-style format verb`
	log.Crit("fatal: %v", "boom")                         // want `log message contains printf-style format verb`
}

func goodMessages() {
	log.Info("started processing")
	log.Error("something failed", "err", "boom")
	log.Warn("100%% complete")
	log.Info("progress: 50%% done")
	log.Debug("no format verbs here", "key", "val")
}
