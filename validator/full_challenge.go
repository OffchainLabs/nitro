//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/pkg/errors"
)

type FullChallengeManager struct {
	inboxTracker          InboxTrackerInterface
	txStreamer            TransactionStreamerInterface
	inboxReader           InboxReaderInterface
	rootChallengeAddr     common.Address
	isExecutionChallenge  bool
	challenge             *ChallengeManager
	blockChallengeBackend *BlockChallengeBackend
	blockChallengeCon     *challengegen.BlockChallenge
	l1Client              bind.ContractBackend
	auth                  *bind.TransactOpts
	blockchain            *core.BlockChain
	startL1Block          uint64
	targetNumMachines     int
}

type InboxReaderInterface interface {
	GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error)
}

func NewFullChallengeManager(
	ctx context.Context,
	inboxTracker InboxTrackerInterface,
	txStreamer TransactionStreamerInterface,
	inboxReader InboxReaderInterface,
	blockchain *core.BlockChain,
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
	blockBackend, err := NewBlockChallengeBackend(ctx, blockchain, inboxTracker, l1Client, challengeAddr)
	if err != nil {
		return nil, err
	}
	blockChallengeManager, err := NewChallengeManager(ctx, l1Client, auth, challengeAddr, startL1Block, blockBackend)
	if err != nil {
		return nil, err
	}
	manager := &FullChallengeManager{
		inboxTracker:          inboxTracker,
		txStreamer:            txStreamer,
		inboxReader:           inboxReader,
		rootChallengeAddr:     challengeAddr,
		isExecutionChallenge:  false,
		challenge:             blockChallengeManager,
		blockChallengeBackend: blockBackend,
		blockChallengeCon:     blockchallengeCon,
		l1Client:              l1Client,
		auth:                  auth,
		blockchain:            blockchain,
		startL1Block:          startL1Block,
		targetNumMachines:     targetNumMachines,
	}
	err = manager.checkForExecutionChallenge(ctx)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *FullChallengeManager) checkForExecutionChallenge(ctx context.Context) error {
	if m.blockChallengeBackend == nil {
		// This has already moved on to an execution challenge
		return nil
	}
	callOpts := &bind.CallOpts{Context: ctx}
	var err error
	callOpts.BlockNumber, err = LatestConfirmedBlock(ctx, m.l1Client)
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
		startHeader := m.blockchain.GetHeaderByHash(GoGlobalStateFromSolidity(startGs).BlockHash)
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
		blockNumU64 := blockNumber.Uint64()
		blockHeader := m.blockchain.GetHeaderByNumber(blockNumU64)
		machine, err := GetZeroStepMachine(ctx)
		if err != nil {
			return err
		}
		message, err := m.txStreamer.GetMessage(blockNumU64)
		if err != nil {
			return err
		}
		nextHeader := m.blockchain.GetHeaderByNumber(blockNumU64 + 1)
		preimages, hasDelayedMsg, delayedMsgNr, err := BlockDataForValidation(m.blockchain, nextHeader, blockHeader, message)
		if err != nil {
			return err
		}
		machine.AddPreimages(preimages)
		globalState, err := m.blockChallengeBackend.FindGlobalStateFromHeader(ctx, blockHeader)
		if err != nil {
			return err
		}
		machine.SetGlobalState(globalState)
		if hasDelayedMsg {
			delayedBytes, err := m.inboxTracker.GetDelayedMessageBytes(delayedMsgNr)
			if err != nil {
				return err
			}
			machine.AddDelayedInboxMessage(delayedMsgNr, delayedBytes)
		}
		batchBytes, err := m.inboxReader.GetSequencerMessageBytes(ctx, globalState.Batch)
		if err != nil {
			return err
		}
		machine.AddSequencerInboxMessage(globalState.Batch, batchBytes)
		execBackend, err := NewExecutionChallengeBackend(machine, m.targetNumMachines, nil)
		if err != nil {
			return err
		}
		newChallenge, err := NewChallengeManager(ctx, m.l1Client, m.auth, addr, m.startL1Block, execBackend)
		if err != nil {
			return err
		}
		m.challenge = newChallenge
		m.blockChallengeBackend = nil
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
