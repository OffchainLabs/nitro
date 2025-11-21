// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package backend

import (
	"context"
	"testing"
)

func TestLocalAnvilLoadAccounts(t *testing.T) {
	a, err := NewAnvilLocal(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.loadAccounts(); err != nil {
		t.Fatal(err)
	}
	if len(a.accounts) == 0 {
		t.Error("No accounts generated")
	}
}
