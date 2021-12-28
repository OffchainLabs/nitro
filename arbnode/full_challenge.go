//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/offchainlabs/arbstate/validator"
	"github.com/pkg/errors"
)

type FullChallengeManager struct {
	rootChallengeAddr    common.Address
	isExecutionChallenge bool
	challenge            *validator.ChallengeManager
	blockChallengeCon    *challengegen.BlockChallenge
	l1Client             bind.ContractBackend
	auth                 *bind.TransactOpts
	node                 *Node
	startL1Block         uint64
	targetNumMachines    int
}

func NewFullChallengeManager(
	ctx context.Context,
	node *Node,
	l1Client bind.ContractBackend,
	auth *bind.TransactOpts,
	challengeAddr common.Address,
	startL1Block uint64,
	targetNumMachines int,
) (*FullChallengeManager, error) {
	blockchallengeCon, err := challengegen.NewBlockChallenge(challengeAddr, l1Client)
	if err != nil {
		return nil, err
	}
	manager := &FullChallengeManager{
		rootChallengeAddr:    challengeAddr,
		isExecutionChallenge: false,
		challenge:            nil,
		blockChallengeCon:    blockchallengeCon,
		l1Client:             l1Client,
		auth:                 auth,
		node:                 node,
		startL1Block:         startL1Block,
		targetNumMachines:    targetNumMachines,
	}
	err = manager.checkForExecutionChallenge(ctx)
	if err != nil {
		return nil, err
	}
	if manager.challenge == nil {
		blockBackend, err := NewBlockChallengeBackend(ctx, node.ArbInterface.BlockChain(), node.InboxTracker, l1Client, challengeAddr)
		if err != nil {
			return nil, err
		}
		manager.challenge, err = validator.NewChallengeManager(ctx, l1Client, auth, challengeAddr, startL1Block, blockBackend)
		if err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func (m *FullChallengeManager) checkForExecutionChallenge(ctx context.Context) error {
	callOpts := &bind.CallOpts{Context: ctx}
	var err error
	callOpts.BlockNumber, err = validator.LatestConfirmedBlock(ctx, m.l1Client)
	if err != nil {
		return err
	}
	addr, err := m.blockChallengeCon.ExecutionChallenge(callOpts)
	if err != nil {
		return err
	}
	if addr != (common.Address{}) {
		startGs, err := m.blockChallengeCon.GetStartGlobalState(callOpts)
		if err != nil {
			return err
		}
		startHeader := m.node.ArbInterface.BlockChain().GetHeaderByHash(GoGlobalStateFromSolidity(startGs).BlockHash)
		if startHeader == nil {
			return errors.New("failed to find challenge start block")
		}
		blockOffset, err := m.blockChallengeCon.ExecutionChallengeAtSteps(callOpts)
		if err != nil {
			return err
		}
		blockNumber := new(big.Int).Add(startHeader.Number, blockOffset)
		if !blockNumber.IsUint64() {
			return errors.New("execution challenge occurred at non-uint64 block number")
		}
		initialMachine, err := m.node.BlockValidator.GetInitialMachineForBlock(ctx, blockNumber.Uint64())
		if err != nil {
			return err
		}
		execBackend, err := validator.NewExecutionChallengeBackend(initialMachine, m.targetNumMachines, nil)
		if err != nil {
			return err
		}
		newChallenge, err := validator.NewChallengeManager(ctx, m.l1Client, m.auth, addr, m.startL1Block, execBackend)
		if err != nil {
			return err
		}
		m.challenge = newChallenge
	}
	return nil
}

func (m *FullChallengeManager) Act(ctx context.Context) (*types.Transaction, error) {
	err := m.checkForExecutionChallenge(ctx)
	if err != nil {
		return nil, err
	}
	return m.challenge.Act(ctx)
}
