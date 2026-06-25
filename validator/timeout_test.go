// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package validator

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// mockNetTimeoutError implements net.Error for testing
type mockNetTimeoutError struct {
	timeout bool
}

func (e *mockNetTimeoutError) Error() string   { return "mock net error" }
func (e *mockNetTimeoutError) Timeout() bool   { return e.timeout }
func (e *mockNetTimeoutError) Temporary() bool { return false }

// mockRPCError implements rpc.Error interface (ErrorCode() int + Error() string)
type mockRPCError struct {
	code    int
	message string
}

func (e *mockRPCError) Error() string  { return e.message }
func (e *mockRPCError) ErrorCode() int { return e.code }

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		// nil and generic errors
		{"nil error", nil, false},
		{"generic error", errors.New("something failed"), false},
		{"validation failed", errors.New("validation failed: got {0x00 0x00 0 0}"), false},

		// Typed: context.DeadlineExceeded
		{"context.DeadlineExceeded", context.DeadlineExceeded, true},
		{"wrapped DeadlineExceeded", fmt.Errorf("validation: %w", context.DeadlineExceeded), true},
		{"context.Canceled is not timeout", context.Canceled, false},

		// Typed: net.Error with Timeout()
		{"net timeout error", &mockNetTimeoutError{timeout: true}, true},
		{"net non-timeout error", &mockNetTimeoutError{timeout: false}, false},

		// Typed: rpc.Error with error code
		{"RPC timeout code -32002", &mockRPCError{code: -32002, message: "request timed out"}, true},
		{"RPC default code -32000", &mockRPCError{code: -32000, message: "some error"}, false},

		// String-based: Redis-serialized errors (type info lost)
		{"string: request timed out", errors.New("request timed out"), true},
		{"string: context deadline exceeded", errors.New("context deadline exceeded"), true},
		{"string: i/o timeout", errors.New("i/o timeout"), true},
		{"string: redis producer timeout", errors.New("error getting response, request has been waiting for too long"), true},

		// String-based: embedded in longer messages
		{"string: wrapped timeout message", errors.New("rpc error: request timed out"), true},
		{"string: wrapped deadline message", errors.New("call failed: context deadline exceeded"), true},
		{"string: wrapped i/o timeout", errors.New("connection error: i/o timeout"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("IsTimeoutError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
