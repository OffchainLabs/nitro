//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

var rollupCreatedID common.Hash
var nodeCreatedID common.Hash
var challengeCreatedID common.Hash

func init() {
	parsedRollup, err := rollupgen.RollupUserLogicMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	rollupCreatedID = parsedRollup.Events["RollupCreated"].ID
	nodeCreatedID = parsedRollup.Events["NodeCreated"].ID
	challengeCreatedID = parsedRollup.Events["RollupChallengeStarted"].ID
}

type StakerInfo struct {
	Index            uint64
	LatestStakedNode uint64
	AmountStaked     *big.Int
	CurrentChallenge *common.Address
}

type RollupWatcher struct {
	address      common.Address
	fromBlock    int64
	client       arbutil.L1Interface
	baseCallOpts bind.CallOpts
	*rollupgen.RollupUserLogic
}

func NewRollupWatcher(address common.Address, fromBlock int64, client arbutil.L1Interface, callOpts bind.CallOpts) (*RollupWatcher, error) {
	con, err := rollupgen.NewRollupUserLogic(address, client)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &RollupWatcher{
		address:         address,
		fromBlock:       fromBlock,
		client:          client,
		baseCallOpts:    callOpts,
		RollupUserLogic: con,
	}, nil
}

func (r *RollupWatcher) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := r.baseCallOpts
	opts.Context = ctx
	return &opts
}

func (r *RollupWatcher) LookupCreation(ctx context.Context) (*rollupgen.RollupUserLogicRollupInitialized, error) {
	var query = ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: big.NewInt(r.fromBlock),
		ToBlock:   big.NewInt(r.fromBlock),
		Addresses: []common.Address{r.address},
		Topics:    [][]common.Hash{{rollupCreatedID}},
	}
	logs, err := r.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(logs) == 0 {
		return nil, errors.New("rollup not created")
	}
	if len(logs) > 1 {
		return nil, errors.New("rollup created multiple times")
	}
	ev, err := r.ParseRollupInitialized(logs[0])
	return ev, errors.WithStack(err)
}

func (r *RollupWatcher) LookupNode(ctx context.Context, number uint64) (*NodeInfo, error) {
	var numberAsHash common.Hash
	binary.BigEndian.PutUint64(numberAsHash[(32-8):], number)
	var query = ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: big.NewInt(r.fromBlock),
		ToBlock:   nil,
		Addresses: []common.Address{r.address},
		Topics:    [][]common.Hash{{nodeCreatedID}, {numberAsHash}},
	}
	logs, err := r.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(logs) == 0 {
		return nil, errors.New("Couldn't find requested node")
	}
	if len(logs) > 1 {
		return nil, errors.New("Found multiple instances of requested node")
	}
	ethLog := logs[0]
	parsedLog, err := r.ParseNodeCreated(ethLog)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &NodeInfo{
		NodeNum:            parsedLog.NodeNum,
		BlockProposed:      ethLog.BlockNumber,
		Assertion:          NewAssertionFromSolidity(parsedLog.Assertion),
		AfterInboxBatchAcc: parsedLog.AfterInboxBatchAcc,
		NodeHash:           parsedLog.NodeHash,
		WasmModuleRoot:     parsedLog.WasmModuleRoot,
	}, nil
}

func (r *RollupWatcher) LookupNodeChildren(ctx context.Context, parentHash [32]byte, fromBlock uint64) ([]*NodeInfo, error) {
	var query = ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   nil,
		Addresses: []common.Address{r.address},
		Topics:    [][]common.Hash{{nodeCreatedID}, nil, {parentHash}},
	}
	logs, err := r.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	infos := make([]*NodeInfo, 0, len(logs))
	for i, ethLog := range logs {
		parsedLog, err := r.ParseNodeCreated(ethLog)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		lastHashIsSibling := [1]byte{0}
		if i > 0 {
			lastHashIsSibling[0] = 1
		}
		infos = append(infos, &NodeInfo{
			NodeNum:            parsedLog.NodeNum,
			BlockProposed:      ethLog.BlockNumber,
			Assertion:          NewAssertionFromSolidity(parsedLog.Assertion),
			AfterInboxBatchAcc: parsedLog.AfterInboxBatchAcc,
			NodeHash:           parsedLog.NodeHash,
		})
	}
	return infos, nil
}

func (r *RollupWatcher) LookupChallengedNode(ctx context.Context, address common.Address) (uint64, error) {
	addressQuery := common.Hash{}
	copy(addressQuery[12:], address.Bytes())

	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: big.NewInt(r.fromBlock),
		ToBlock:   nil,
		Addresses: []common.Address{r.address},
		Topics:    [][]common.Hash{{challengeCreatedID}, {addressQuery}},
	}
	logs, err := r.client.FilterLogs(ctx, query)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	if len(logs) == 0 {
		return 0, errors.New("no matching challenge")
	}

	if len(logs) > 1 {
		return 0, errors.New("too many matching challenges")
	}

	challenge, err := r.ParseRollupChallengeStarted(logs[0])
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return challenge.ChallengedNode, nil
}

func (r *RollupWatcher) StakerInfo(ctx context.Context, staker common.Address) (*StakerInfo, error) {
	info, err := r.StakerMap(r.getCallOpts(ctx), staker)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !info.IsStaked {
		return nil, nil
	}
	stakerInfo := &StakerInfo{
		Index:            info.Index,
		LatestStakedNode: info.LatestStakedNode,
		AmountStaked:     info.AmountStaked,
	}
	emptyAddress := common.Address{}
	if info.CurrentChallenge != emptyAddress {
		chal := info.CurrentChallenge
		stakerInfo.CurrentChallenge = &chal
	}
	return stakerInfo, nil
}
