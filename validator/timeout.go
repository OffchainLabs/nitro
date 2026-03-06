// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package validator

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/pubsub"
)

const (
	ioTimeoutMessage = "i/o timeout" // net package timeout
)

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

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var rpcErr rpc.Error
	if errors.As(err, &rpcErr) && rpcErr.ErrorCode() == rpc.ErrcodeTimeout {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// String-based detection

	errMsg := err.Error()
	if strings.Contains(errMsg, pubsub.TimeoutErrorMessage) ||
		strings.Contains(errMsg, rpc.ErrMsgTimeout) ||
		strings.Contains(errMsg, context.DeadlineExceeded.Error()) ||
		strings.Contains(errMsg, ioTimeoutMessage) {
		return true
	}

	return false
}
