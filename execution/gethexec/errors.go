// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import "github.com/ethereum/go-ethereum/common"

const (
	errCodeTxSyncTimeout = 4
)

type txSyncTimeoutError struct {
	msg  string
	hash common.Hash
}

func (e *txSyncTimeoutError) Error() string          { return e.msg }
func (e *txSyncTimeoutError) ErrorCode() int         { return errCodeTxSyncTimeout }
func (e *txSyncTimeoutError) ErrorData() interface{} { return e.hash.Hex() }
