// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package backend

import (
	"context"
	"testing"
	"time"
)

func TestLocalAnvilLoadAccounts(t *testing.T) {
	a, err := NewAnvilLocal(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.loadAccounts(); err != nil {
		t.Fatal(err)
	}
	if a.alice == nil {
		t.Error("Alice is nil")
	}
	if a.bob == nil {
		t.Error("Bob is nil")
	}
	if a.charlie == nil {
		t.Error("Charlie is nil")
	}
	if a.deployer == nil {
		t.Error("Deployer is nil")
	}
}

func TestLocalAnvilStarts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	a, err := NewAnvilLocal(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.DeployRollup(); err != nil {
		t.Fatal(err)
	}

	// There should be at least 100 blocks
	bn, err2 := a.Client().BlockNumber(ctx)
	if err2 != nil {
		t.Fatal(err2)
	}
	if bn < 100 {
		t.Errorf("Expected at least 100 blocks at start, but got only %d", bn)
	}

	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
}
