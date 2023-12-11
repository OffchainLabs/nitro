// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package headerreader

import (
	"io"
	"testing"
)

func TestExecutionRevertedRegexp(t *testing.T) {
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
		if !ExecutionRevertedRegexp.MatchString(errString) {
			t.Fatalf("execution reverted regexp didn't match %q", errString)
		}
	}
	// This regexp should not match random IO errors
	if ExecutionRevertedRegexp.MatchString(io.ErrUnexpectedEOF.Error()) {
		t.Fatal("execution reverted regexp matched unexpected EOF")
	}
}
