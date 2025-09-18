// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package backend

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/setup"
)

var _ Backend = &LocalSimulatedBackend{}

type LocalSimulatedBackend struct {
	blockTime time.Duration
	setup     *setup.ChainSetup
}

func (l *LocalSimulatedBackend) Start(ctx context.Context) error {
	// Advance blocks in the background.
	go func() {
		ticker := time.NewTicker(l.blockTime)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				l.setup.Backend.Commit()
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (l *LocalSimulatedBackend) Stop(ctx context.Context) error {
	return nil
}

func (l *LocalSimulatedBackend) Client() protocol.ChainBackend {
	return l.setup.Backend
}

func (l *LocalSimulatedBackend) Commit() common.Hash {
	return l.setup.Backend.Commit()
}

func (l *LocalSimulatedBackend) Accounts() []*bind.TransactOpts {
	accs := make([]*bind.TransactOpts, len(l.setup.Accounts))
	for i := 0; i < len(l.setup.Accounts); i++ {
		accs[i] = l.setup.Accounts[i].TxOpts
	}
	return accs
}

func (l *LocalSimulatedBackend) ContractAddresses() *setup.RollupAddresses {
	return l.setup.Addrs
}

func (l *LocalSimulatedBackend) DeployRollup(_ context.Context, _ ...challenge_testing.Opt) (*setup.RollupAddresses, error) {
	// No-op, as the sim backend deploys the rollup on initialization.
	return l.setup.Addrs, nil
}

func NewSimulated(blockTime time.Duration, opts ...setup.Opt) (*LocalSimulatedBackend, error) {
	setup, err := setup.ChainsWithEdgeChallengeManager(opts...)
	if err != nil {
		return nil, err
	}
	return &LocalSimulatedBackend{blockTime: blockTime, setup: setup}, nil
}
