// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package backend

import (
	"context"

	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Backend interface {
	// Start sets up the backend and waits until the process is in a ready state.
	Start(ctx context.Context) error
	// Client returns the backend's client.
	Client() setup.Backend
	// Accounts managed by the backend.
	Accounts() []*bind.TransactOpts
	// DeployRollup contract, if not already deployed.
	DeployRollup(ctx context.Context, opts ...challenge_testing.Opt) (*setup.RollupAddresses, error)
	// Contract addresses relevant to the challenge protocol.
	ContractAddresses() *setup.RollupAddresses
	// Commit a tx to the backend, if possible (simulated backend requires this)
	Commit() common.Hash
}
