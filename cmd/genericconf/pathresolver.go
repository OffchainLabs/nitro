// Copyright 2026-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package genericconf

import (
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
)

func DefaultPathResolver(workdir string) func(string) string {
	if workdir == "" {
		var err error
		workdir, err = os.Getwd()
		if err != nil {
			log.Warn("Failed to get workdir", "err", err)
		}
	}
	return func(path string) string {
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(workdir, path)
	}
}
