// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package logfmt

import (
	"fmt"

	"go-ethereum/log"
)

func badFormatVerbs() {
	log.Warn("failed init redis for %v: %w", "mr", "err") // want `log message contains printf-style format verb`
	log.Error("count is %d", 5)                           // want `log message contains printf-style format verb`
	log.Info("hash: %s", "abc")                           // want `log message contains printf-style format verb`
	log.Debug("value is %+v", "x")                        // want `log message contains printf-style format verb`
	log.Trace("hex: %02x", 255)                           // want `log message contains printf-style format verb`
	log.Crit("fatal: %v", "boom")                         // want `log message contains printf-style format verb`
	log.Info("width: %*d", "key", 5)                      // want `log message contains printf-style format verb`
	log.Info("left: %-10s", "key", "v")                   // want `log message contains printf-style format verb`
	log.Info("alt: %#v", "key", "v")                      // want `log message contains printf-style format verb`
	log.Info("quoted: %q", "key", "v")                    // want `log message contains printf-style format verb`
	log.Info("float: %.2f", "key", 3.14)                  // want `log message contains printf-style format verb`
}

func loggerMethod() {
	logger := log.Root()
	logger.Warn("value: %v", "x") // want `log message contains printf-style format verb`
}

// Known false positive: %d appears in prose without a space separator.
func knownFalsePositive() {
	log.Info("50%done") // want `log message contains printf-style format verb`
}

func goodMessages() {
	log.Info("started processing")
	log.Error("something failed", "err", "boom")
	log.Warn("100% complete")
	log.Info("progress: 50% done")
	log.Debug("no format verbs here", "key", "val")
}

// Non-geth type with matching method name -- must NOT be flagged.
type myLogger struct{}

func (myLogger) Error(msg string, args ...interface{}) {}

func nonGethLogger() {
	l := myLogger{}
	l.Error("value is %d", 5)
}

func nonLiteralArg() {
	msg := fmt.Sprintf("value is %d", 5)
	log.Info(msg)
}
