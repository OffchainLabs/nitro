// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package backend

import (
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Backend interface {
	// Operations
	// Start sets up the backend and waits until the process is in a ready state.
	Start() error
	// Stop tears down the backend.
	Stop() error
	// Client returns the backend's client.
	Client() *ethclient.Client

	// Transactors
	// Alice represents the transactor for Alice's account.
	Alice() *bind.TransactOpts
	// Bob represents the transactor for Bob's account.
	Bob() *bind.TransactOpts
	// Charlie represents the transactor for Charlie's account.
	Charlie() *bind.TransactOpts

	// Deployer functions
	// DeployRollup contract, if not already deployed.
	DeployRollup() (common.Address, error)

	// Contract addresses relevant to the challenge protocol.
	ContractAddresses() *setup.RollupAddresses
}
