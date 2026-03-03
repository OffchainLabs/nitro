// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package validator

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

const rpcTimeoutErrorCode = -32002 // errcodeTimeout from go-ethereum/rpc/errors.go

// IsTimeoutError returns true if the error represents a timeout condition from
// any validation path: direct RPC, JIT machine TCP, or Redis producer.
//
// It uses typed error detection when possible (for errors that preserve Go type
// information) and falls back to string matching for errors that have been
// serialized through Redis (which loses all type information).
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Typed detection

	// 1. Context deadline exceeded (RPC client-side timeout, validation context timeout)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// 2. RPC server timeout: jsonError with code -32002 ("request timed out")
	var rpcErr rpc.Error
	if errors.As(err, &rpcErr) && rpcErr.ErrorCode() == rpcTimeoutErrorCode {
		return true
	}

	// 3. Network timeout (JIT machine TCP deadline exceeded)
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// String-based detection

	errMsg := err.Error()

	// Redis producer timeout (pubsub/producer.go)
	if strings.Contains(errMsg, "request has been waiting for too long") {
		return true
	}

	// RPC server timeout message (go-ethereum/rpc/errors.go errMsgTimeout)
	if strings.Contains(errMsg, "request timed out") {
		return true
	}

	// context.DeadlineExceeded serialized as string
	if strings.Contains(errMsg, "context deadline exceeded") {
		return true
	}

	// Network I/O timeout serialized as string
	if strings.Contains(errMsg, "i/o timeout") {
		return true
	}

	return false
}
