// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package headerreader

import (
	"errors"
	"io"
	"testing"
)

func TestExecutionReverted(t *testing.T) {
	executionRevertedErrors := []string{
		// go-ethereum and most other execution clients return "execution reverted"
		"execution reverted",
		// execution clients may decode the EVM revert data as a string and include it in the error
		"execution reverted: FOO",
		// besu returns "Execution reverted"
		"Execution reverted",
		// nethermind returns "VM execution error."
		"VM execution error.",
	}
	for _, errString := range executionRevertedErrors {
		if !IsExecutionReverted(errors.New(errString)) {
			t.Fatalf("execution reverted regexp didn't match %q", errString)
		}
	}
	// This regexp should not match random IO errors
	if IsExecutionReverted(errors.New(io.ErrUnexpectedEOF.Error())) {
		t.Fatal("execution reverted regexp matched unexpected EOF")
	}

	if !IsExecutionReverted(&executionRevertedError{}) {
		t.Fatal("execution reverted error didn't match")
	}
}

type executionRevertedError struct{}

func (e *executionRevertedError) ErrorCode() int { return 3 }

func (e *executionRevertedError) Error() string {
	return "executionRevertedError"
}
