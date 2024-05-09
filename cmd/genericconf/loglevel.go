// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package genericconf

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/slog"
)

func ToSlogLevel(str string) (slog.Level, error) {
	switch strings.ToLower(str) {
	case "trace":
		return log.LevelTrace, nil
	case "debug":
		return log.LevelDebug, nil
	case "info":
		return log.LevelInfo, nil
	case "warn":
		return log.LevelWarn, nil
	case "error":
		return log.LevelError, nil
	case "crit":
		return log.LevelCrit, nil
	default:
		legacyLevel, err := strconv.Atoi(str)
		if err != nil {
			// Leave legacy geth numeric log levels undocumented, but if anyone happens
			// to be using them, it will work.
			return log.LevelTrace, errors.New("invalid log-level")
		}
		return log.FromLegacyLevel(legacyLevel), nil
	}
}
