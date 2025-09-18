// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package backend

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/setup"
)

type Backend interface {
	// Start sets up the backend and waits until the process is in a ready state.
	Start(ctx context.Context) error
	// Client returns the backend's client.
	Client() protocol.ChainBackend
	// Accounts managed by the backend.
	Accounts() []*bind.TransactOpts
	// DeployRollup contract, if not already deployed.
	DeployRollup(ctx context.Context, opts ...challenge_testing.Opt) (*setup.RollupAddresses, error)
	// Contract addresses relevant to the challenge protocol.
	ContractAddresses() *setup.RollupAddresses
	// Commit a tx to the backend, if possible (simulated backend requires this)
	Commit() common.Hash
}
