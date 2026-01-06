// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import (
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/genericconf"
)

func SetLogger(logLevelStr string, logType string) error {
	logLevel, err := genericconf.ToSlogLevel(logLevelStr)
	if err != nil {
		return err
	}

	handler, err := genericconf.HandlerFromLogType(logType, io.Writer(os.Stderr))
	if err != nil {
		return fmt.Errorf("error parsing log type when creating handler: %w", err)
	}
	glogger := log.NewGlogHandler(handler)
	glogger.Verbosity(logLevel)
	log.SetDefault(log.NewLogger(glogger))

	return nil
}
