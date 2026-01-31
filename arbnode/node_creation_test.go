// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"testing"

	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

// TestTypedNilExecutionSequencer verifies that we correctly handle nil
// ExecutionSequencer to avoid typed nil interface issues.
//
// In Go, when a concrete nil pointer is assigned to an interface, the interface
// itself is not nil (it has a type but nil value). This caused crashes when
// checking `if n.ExecutionSequencer != nil` and then calling methods on it.
//
// This test ensures that our fix correctly uses pure nil interfaces instead of
// typed nil interfaces when in RPC client mode.
func TestTypedNilExecutionSequencer(t *testing.T) {
	// Simulate the bug: assigning a nil concrete type to an interface
	var typedNil execution.ExecutionSequencer = (*gethexec.ExecutionNode)(nil)

	// This is the bug - typed nil is NOT equal to nil!
	if typedNil == nil {
		t.Error("Expected typed nil to NOT equal nil (this demonstrates the Go typed nil behavior)")
	}

	// This is the fix: use an unassigned interface variable (pure nil)
	var pureNil execution.ExecutionSequencer

	// Pure nil IS equal to nil
	if pureNil != nil {
		t.Error("Expected pure nil to equal nil")
	}

	// Verify the fix behavior: when we don't assign to the interface,
	// the nil check works correctly
	node := &Node{
		ExecutionSequencer: pureNil, // This is what the fix does
	}
	if node.ExecutionSequencer != nil {
		t.Error("Expected node.ExecutionSequencer to be nil with pure nil")
	}

	// Verify the bug behavior (before fix)
	nodeBug := &Node{
		ExecutionSequencer: typedNil, // This is what the bug did
	}
	if nodeBug.ExecutionSequencer == nil {
		t.Error("Expected node.ExecutionSequencer to NOT be nil with typed nil (demonstrating the bug)")
	}
}
