//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
)

type ExecutionChallengeManager struct {
	con           *challengegen.ExecutionChallenge
	client        bind.ContractBackend
	challengeAddr common.Address
	auth          *bind.TransactOpts
	actingAs      common.Address
	cache         *MachineCache
}

func NewExecutionChallengeManager(ctx context.Context, client bind.ContractBackend, auth *bind.TransactOpts, addr common.Address, initialMachine *ArbitratorMachine, targetNumMachines int) (*ExecutionChallengeManager, error) {
	con, err := challengegen.NewExecutionChallenge(addr, client)
	if err != nil {
		return nil, err
	}
	cache, err := NewMachineCache(ctx, initialMachine, targetNumMachines)
	if err != nil {
		return nil, err
	}
	return &ExecutionChallengeManager{
		con:           con,
		client:        client,
		challengeAddr: addr,
		auth:          auth,
		actingAs:      auth.From,
		cache:         cache,
	}, nil
}
