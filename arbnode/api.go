// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/validator"
	"github.com/pkg/errors"
)

type BlockValidatorAPI struct {
	val        *validator.BlockValidator
	blockchain *core.BlockChain
}

func (a *BlockValidatorAPI) RevalidateBlock(ctx context.Context, blockNum rpc.BlockNumberOrHash, moduleRootOptional *common.Hash) (bool, error) {
	header, err := arbitrum.HeaderByNumberOrHash(a.blockchain, blockNum)
	if err != nil {
		return false, err
	}
	var moduleRoot common.Hash
	if moduleRootOptional != nil {
		moduleRoot = *moduleRootOptional
	} else {
		moduleRoots := a.val.GetModuleRootsToValidate()
		if len(moduleRoots) == 0 {
			return false, errors.New("no current WasmModuleRoot configured, must provide parameter")
		}
		moduleRoot = moduleRoots[0]
	}
	return a.val.ValidateBlock(ctx, header, moduleRoot)
}

func (a *BlockValidatorAPI) LatestValidatedBlock(ctx context.Context) (hexutil.Uint64, error) {
	block := a.val.LastBlockValidated()
	return hexutil.Uint64(block), nil
}

func (a *BlockValidatorAPI) LatestValidatedBlockHash(ctx context.Context) (common.Hash, error) {
	_, hash, _ := a.val.LastBlockValidatedAndHash()
	return hash, nil
}

type ArbDebugAPI struct {
	blockchain *core.BlockChain
}

type PricingModelHistory struct {
	First            uint64     `json:"first"`
	Timestamp        []uint64   `json:"timestamp"`
	BaseFee          []*big.Int `json:"baseFee"`
	GasBacklog       []uint64   `json:"gasBacklog"`
	GasUsed          []uint64   `json:"gasUsed"`
	MinBaseFee       *big.Int   `json:"minBaseFee"`
	SpeedLimit       uint64     `json:"speedLimit"`
	PerBlockGasLimit uint64     `json:"perBlockGasLimit"`
	PricingInertia   uint64     `json:"pricingInertia"`
	BacklogTolerance uint64     `json:"backlogTolerance"`

	L1BaseFeeEstimate    []*big.Int `json:"l1BaseFeeEstimate"`
	L1LastSurplus        []*big.Int `json:"l1LastSurplus"`
	L1FundsDue           []*big.Int `json:"l1FundsDue"`
	L1FundsDueForRewards []*big.Int `json:"l1FundsDueForRewards"`
	L1UnitsSinceUpdate   []uint64   `json:"l1UnitsSinceUpdate"`
	L1LastUpdateTime     []uint64   `json:"l1LastUpdateTime"`
	L1EquilibrationUnits *big.Int   `json:"l1EquilibrationUnits"`
	L1PricingInertia     uint64     `json:"l1PricingInertia"`
	L1PerUnitReward      uint64     `json:"l1PerUnitReward"`
	L1PayRewardTo        string     `json:"l1PayRewardTo"`
}

func (api *ArbDebugAPI) PricingModel(ctx context.Context, start, end rpc.BlockNumber) (PricingModelHistory, error) {
	start, _ = api.blockchain.ClipToPostNitroGenesis(start)
	end, _ = api.blockchain.ClipToPostNitroGenesis(end)

	blocks := end.Int64() - start.Int64() + 1
	if blocks <= 0 {
		return PricingModelHistory{}, fmt.Errorf("invalid block range: %v to %v", start.Int64(), end.Int64())
	}

	history := PricingModelHistory{
		First:                uint64(start),
		Timestamp:            make([]uint64, blocks),
		BaseFee:              make([]*big.Int, blocks),
		GasBacklog:           make([]uint64, blocks),
		GasUsed:              make([]uint64, blocks),
		L1BaseFeeEstimate:    make([]*big.Int, blocks),
		L1LastSurplus:        make([]*big.Int, blocks),
		L1FundsDue:           make([]*big.Int, blocks),
		L1FundsDueForRewards: make([]*big.Int, blocks),
		L1UnitsSinceUpdate:   make([]uint64, blocks),
		L1LastUpdateTime:     make([]uint64, blocks),
	}

	for i := uint64(0); i < uint64(blocks); i++ {
		state, header, err := stateAndHeader(api.blockchain, i+uint64(start))
		if err != nil {
			return history, err
		}
		l1Pricing := state.L1PricingState()
		l2Pricing := state.L2PricingState()

		history.Timestamp[i] = header.Time
		history.BaseFee[i] = header.BaseFee

		gasBacklog, _ := l2Pricing.GasBacklog()
		l1BaseFeeEstimate, _ := l1Pricing.PricePerUnit()
		l1FundsDue, _ := l1Pricing.BatchPosterTable().TotalFundsDue()
		l1FundsDueForRewards, _ := l1Pricing.FundsDueForRewards()
		l1UnitsSinceUpdate, _ := l1Pricing.UnitsSinceUpdate()
		l1LastUpdateTime, _ := l1Pricing.LastUpdateTime()
		l1LastSurplus, _ := l1Pricing.LastSurplus()

		history.GasBacklog[i] = gasBacklog
		history.GasUsed[i] = header.GasUsed

		history.L1BaseFeeEstimate[i] = l1BaseFeeEstimate
		history.L1FundsDue[i] = l1FundsDue
		history.L1FundsDueForRewards[i] = l1FundsDueForRewards
		history.L1UnitsSinceUpdate[i] = l1UnitsSinceUpdate
		history.L1LastUpdateTime[i] = l1LastUpdateTime
		history.L1LastSurplus[i] = l1LastSurplus

		if i == uint64(blocks)-1 {
			speedLimit, _ := l2Pricing.SpeedLimitPerSecond()
			perBlockGasLimit, _ := l2Pricing.PerBlockGasLimit()
			minBaseFee, _ := l2Pricing.MinBaseFeeWei()
			pricingInertia, _ := l2Pricing.PricingInertia()
			backlogTolerance, _ := l2Pricing.BacklogTolerance()

			l1PricingInertia, _ := l1Pricing.Inertia()
			l1EquilibrationUnits, _ := l1Pricing.EquilibrationUnits()
			l1PerUnitReward, _ := l1Pricing.PerUnitReward()
			l1PayRewardsTo, err := l1Pricing.PayRewardsTo()

			if err != nil {
				return history, err
			}
			history.MinBaseFee = minBaseFee
			history.SpeedLimit = speedLimit
			history.PerBlockGasLimit = perBlockGasLimit
			history.PricingInertia = pricingInertia
			history.BacklogTolerance = backlogTolerance

			history.L1PricingInertia = l1PricingInertia
			history.L1EquilibrationUnits = l1EquilibrationUnits
			history.L1PerUnitReward = l1PerUnitReward
			history.L1PayRewardTo = l1PayRewardsTo.Hex()
		}
	}

	return history, nil
}

func (api *ArbDebugAPI) TimeoutQueueHistory(ctx context.Context, start, end rpc.BlockNumber) ([]uint64, error) {
	start, _ = api.blockchain.ClipToPostNitroGenesis(start)
	end, _ = api.blockchain.ClipToPostNitroGenesis(end)

	blocks := end.Int64() - start.Int64() + 1
	if blocks <= 0 {
		return []uint64{}, fmt.Errorf("invalid block range: %v to %v", start.Int64(), end.Int64())
	}

	history := make([]uint64, blocks)

	for i := uint64(0); i < uint64(blocks); i++ {
		state, _, err := stateAndHeader(api.blockchain, i+uint64(start))
		if err != nil {
			return history, err
		}
		size, err := state.RetryableState().TimeoutQueue.Size()
		if err != nil {
			return history, err
		}
		history[i] = size
	}

	return history, nil
}

type TimeoutQueue struct {
	BlockNumber uint64        `json:"blockNumber"`
	Tickets     []common.Hash `json:"tickets"`
	Timeouts    []uint64      `json:"timeouts"`
}

func (api *ArbDebugAPI) TimeoutQueue(ctx context.Context, blockNum rpc.BlockNumber) (TimeoutQueue, error) {

	blockNum, _ = api.blockchain.ClipToPostNitroGenesis(blockNum)

	queue := TimeoutQueue{
		BlockNumber: uint64(blockNum),
		Tickets:     []common.Hash{},
		Timeouts:    []uint64{},
	}

	state, _, err := stateAndHeader(api.blockchain, uint64(blockNum))
	if err != nil {
		return queue, err
	}

	closure := func(index uint64, ticket common.Hash) error {

		// we don't care if the retryable has expired
		retryable, err := state.RetryableState().OpenRetryable(ticket, 0)
		if err != nil {
			return err
		}
		if retryable == nil {
			queue.Tickets = append(queue.Tickets, ticket)
			queue.Timeouts = append(queue.Timeouts, 0)
			return nil
		}
		timeout, err := retryable.CalculateTimeout()
		if err != nil {
			return err
		}
		windows, err := retryable.TimeoutWindowsLeft()
		if err != nil {
			return err
		}
		timeout -= windows * retryables.RetryableLifetimeSeconds

		queue.Tickets = append(queue.Tickets, ticket)
		queue.Timeouts = append(queue.Timeouts, timeout)
		return nil
	}

	err = state.RetryableState().TimeoutQueue.ForEach(closure)
	return queue, err
}

func stateAndHeader(blockchain *core.BlockChain, block uint64) (*arbosState.ArbosState, *types.Header, error) {
	header := blockchain.GetHeaderByNumber(block)
	statedb, err := blockchain.StateAt(header.Root)
	if err != nil {
		return nil, nil, err
	}
	state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	return state, header, err
}
