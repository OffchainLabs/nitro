// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package lifecycle

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
)

const maxConsecutiveIntermittentErrors = 20

func MaybeLogError(err error, s *state.LockedInternalState, tx *storage.QueuedTransaction, msg string) {
	nonce := tx.FullTx.Nonce()
	if err == nil {
		delete(s.ErrorCount, nonce)
		return
	}
	logLevel := log.Error
	isStorageRace := errors.Is(err, storage.ErrStorageRace)
	isFutureReplacePending := strings.Contains(err.Error(), legacypool.ErrFutureReplacePending.Error())
	isNonceTooHigh := strings.Contains(err.Error(), core.ErrNonceTooHigh.Error())
	if isStorageRace || isFutureReplacePending || isNonceTooHigh {
		s.ErrorCount[nonce]++
		if s.ErrorCount[nonce] <= maxConsecutiveIntermittentErrors {
			if isStorageRace {
				logLevel = log.Debug
			} else {
				logLevel = log.Info
			}
		} else if isStorageRace {
			logLevel = log.Warn
		}
	} else {
		delete(s.ErrorCount, nonce)
	}
	logLevel(msg, "err", err, "nonce", nonce, "feeCap", tx.FullTx.GasFeeCap(), "tipCap", tx.FullTx.GasTipCap(), "blobFeeCap", tx.FullTx.BlobGasFeeCap(), "gas", tx.FullTx.Gas())
}
