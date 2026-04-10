// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package execution

import "errors"

// Application-level JSON-RPC error codes for Consensus/Execution communication.
// These are below -32768, outside the JSON-RPC 2.0 spec reserved range
// (-32768 to -32000), so they cannot collide with standard or go-ethereum codes.
const (
	ErrCodeRetrySequencer  = -33000
	ErrCodeInsertLockTaken = -33001
	ErrCodeResultNotFound  = -33002
)

// RPCError is an error that carries a JSON-RPC error code.
//
// On the server side it implements the rpc.Error interface, which causes
// go-ethereum's framework to include the code in the JSON-RPC error response
// instead of using the default -32000.
//
// On the client side the Is method enables code-based comparison via
// errors.Is, so callers do not need to inspect error message strings.
type RPCError struct {
	code int
	msg  string
}

func (e *RPCError) Error() string  { return e.msg }
func (e *RPCError) ErrorCode() int { return e.code }

// Is reports whether target is an RPCError with the same code.
// This makes errors.Is(receivedErr, sentinel) return true whenever the
// received error carries the same code as the sentinel, regardless of whether
// it is the exact same pointer or a jsonError reconstructed from the wire.
func (e *RPCError) Is(target error) bool {
	var t *RPCError
	if errors.As(target, &t) {
		return t.code == e.code
	}
	return false
}
