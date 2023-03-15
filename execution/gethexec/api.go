// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/pkg/errors"
)

type ArbAPI struct {
	txPublisher TransactionPublisher
}

func NewArbAPI(publisher TransactionPublisher) *ArbAPI {
	return &ArbAPI{publisher}
}

func (a *ArbAPI) CheckPublisherHealth(ctx context.Context) error {
	return a.txPublisher.CheckHealth(ctx)
}

type ArbDebugAPI struct {
	blockchain        *core.BlockChain
	blockRangeBound   uint64
	timeoutQueueBound uint64
}

func NewArbDebugAPI(blockchain *core.BlockChain, blockRangeBound uint64, timeoutQueueBound uint64) *ArbDebugAPI {
	return &ArbDebugAPI{blockchain, blockRangeBound, timeoutQueueBound}
}

type PricingModelHistory struct {
	Start            uint64     `json:"start"`
	End              uint64     `json:"end"`
	Step             uint64     `json:"step"`
	Timestamp        []uint64   `json:"timestamp"`
	BaseFee          []*big.Int `json:"baseFee"`
	GasBacklog       []uint64   `json:"gasBacklog"`
	GasUsed          []uint64   `json:"gasUsed"`
	MinBaseFee       *big.Int   `json:"minBaseFee"`
	SpeedLimit       uint64     `json:"speedLimit"`
	PerBlockGasLimit uint64     `json:"perBlockGasLimit"`
	PricingInertia   uint64     `json:"pricingInertia"`
	BacklogTolerance uint64     `json:"backlogTolerance"`

	L1BaseFeeEstimate      []*big.Int `json:"l1BaseFeeEstimate"`
	L1LastSurplus          []*big.Int `json:"l1LastSurplus"`
	L1FundsDue             []*big.Int `json:"l1FundsDue"`
	L1FundsDueForRewards   []*big.Int `json:"l1FundsDueForRewards"`
	L1UnitsSinceUpdate     []uint64   `json:"l1UnitsSinceUpdate"`
	L1LastUpdateTime       []uint64   `json:"l1LastUpdateTime"`
	L1EquilibrationUnits   *big.Int   `json:"l1EquilibrationUnits"`
	L1PerBatchCost         int64      `json:"l1PerBatchCost"`
	L1AmortizedCostCapBips uint64     `json:"l1AmortizedCostCapBips"`
	L1PricingInertia       uint64     `json:"l1PricingInertia"`
	L1PerUnitReward        uint64     `json:"l1PerUnitReward"`
	L1PayRewardTo          string     `json:"l1PayRewardTo"`
}

func (api *ArbDebugAPI) evenlySpaceBlocks(start, end rpc.BlockNumber) (uint64, uint64, uint64, uint64, error) {
	start, _ = api.blockchain.ClipToPostNitroGenesis(start)
	end, _ = api.blockchain.ClipToPostNitroGenesis(end)

	blocks := end.Int64() - start.Int64() + 1
	bound := int64(api.blockRangeBound)
	step := int64(1)
	if blocks > bound {
		step = int64(float64(blocks)/float64(bound) + 0.5)
		blocks = arbmath.MinInt(bound, blocks/step)
	}
	if blocks <= 0 {
		return 0, 0, 0, 0, fmt.Errorf("invalid block range: %v to %v", start.Int64(), end.Int64())
	}

	first := uint64(end.Int64() - step*(blocks-1)) // minus 1 to include the fact that we start from the last
	return first, uint64(step), uint64(end), uint64(blocks), nil
}

func (api *ArbDebugAPI) PricingModel(ctx context.Context, start, end rpc.BlockNumber) (PricingModelHistory, error) {

	first, step, last, blocks, err := api.evenlySpaceBlocks(start, end)
	if err != nil {
		return PricingModelHistory{}, err
	}

	history := PricingModelHistory{
		Start:                first,
		End:                  last,
		Step:                 step,
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

	for i := uint64(0); i < blocks; i++ {
		state, header, err := stateAndHeader(api.blockchain, first+i*step)
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
			l1PerBatchCost, _ := l1Pricing.PerBatchGasCost()
			l1AmortizedCostCapBips, _ := l1Pricing.AmortizedCostCapBips()
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
			history.L1PerBatchCost = l1PerBatchCost
			history.L1AmortizedCostCapBips = l1AmortizedCostCapBips
			history.L1PerUnitReward = l1PerUnitReward
			history.L1PayRewardTo = l1PayRewardsTo.Hex()
		}
	}
	return history, nil
}

type TimeoutQueueHistory struct {
	Start uint64   `json:"start"`
	End   uint64   `json:"end"`
	Step  uint64   `json:"step"`
	Count []uint64 `json:"count"`
}

func (api *ArbDebugAPI) TimeoutQueueHistory(ctx context.Context, start, end rpc.BlockNumber) (TimeoutQueueHistory, error) {
	first, step, last, blocks, err := api.evenlySpaceBlocks(start, end)
	if err != nil {
		return TimeoutQueueHistory{}, err
	}

	history := TimeoutQueueHistory{
		Start: first,
		End:   last,
		Step:  step,
		Count: make([]uint64, blocks),
	}

	for i := uint64(0); i < blocks; i++ {
		state, _, err := stateAndHeader(api.blockchain, first+i*step)
		if err != nil {
			return history, err
		}
		size, err := state.RetryableState().TimeoutQueue.Size()
		if err != nil {
			return history, err
		}
		history.Count[i] = size
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

	closure := func(index uint64, ticket common.Hash) (bool, error) {

		// we don't care if the retryable has expired
		retryable, err := state.RetryableState().OpenRetryable(ticket, 0)
		if err != nil {
			return false, err
		}
		if retryable == nil {
			queue.Tickets = append(queue.Tickets, ticket)
			queue.Timeouts = append(queue.Timeouts, 0)
			return false, nil
		}
		timeout, err := retryable.CalculateTimeout()
		if err != nil {
			return false, err
		}
		windows, err := retryable.TimeoutWindowsLeft()
		if err != nil {
			return false, err
		}
		timeout -= windows * retryables.RetryableLifetimeSeconds

		queue.Tickets = append(queue.Tickets, ticket)
		queue.Timeouts = append(queue.Timeouts, timeout)
		return index == api.timeoutQueueBound, nil
	}

	err = state.RetryableState().TimeoutQueue.ForEach(closure)
	return queue, err
}

func stateAndHeader(blockchain *core.BlockChain, block uint64) (*arbosState.ArbosState, *types.Header, error) {
	header := blockchain.GetHeaderByNumber(block)
	if !blockchain.Config().IsArbitrumNitro(header.Number) {
		return nil, nil, types.ErrUseFallback
	}
	statedb, err := blockchain.StateAt(header.Root)
	if err != nil {
		return nil, nil, err
	}
	state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	return state, header, err
}

type ArbTraceForwarderAPI struct {
	fallbackClientUrl     string
	fallbackClientTimeout time.Duration

	initialized    int32
	mutex          sync.Mutex
	fallbackClient types.FallbackClient
}

func NewArbTraceForwarderAPI(fallbackClientUrl string, fallbackClientTimeout time.Duration) *ArbTraceForwarderAPI {
	return &ArbTraceForwarderAPI{
		fallbackClientUrl:     fallbackClientUrl,
		fallbackClientTimeout: fallbackClientTimeout,
	}
}

func (api *ArbTraceForwarderAPI) getFallbackClient() (types.FallbackClient, error) {
	if atomic.LoadInt32(&api.initialized) == 1 {
		return api.fallbackClient, nil
	}
	api.mutex.Lock()
	defer api.mutex.Unlock()
	if atomic.LoadInt32(&api.initialized) == 1 {
		return api.fallbackClient, nil
	}
	fallbackClient, err := arbitrum.CreateFallbackClient(api.fallbackClientUrl, api.fallbackClientTimeout)
	if err != nil {
		return nil, err
	}
	api.fallbackClient = fallbackClient
	atomic.StoreInt32(&api.initialized, 1)
	return api.fallbackClient, nil
}

func (api *ArbTraceForwarderAPI) forward(ctx context.Context, method string, args ...interface{}) (*json.RawMessage, error) {
	fallbackClient, err := api.getFallbackClient()
	if err != nil {
		return nil, err
	}
	if fallbackClient == nil {
		return nil, errors.New("arbtrace calls forwarding not configured") // TODO(magic)
	}
	var resp *json.RawMessage
	err = fallbackClient.CallContext(ctx, &resp, method, args...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (api *ArbTraceForwarderAPI) Call(ctx context.Context, callArgs json.RawMessage, traceTypes json.RawMessage, blockNum json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_call", callArgs, traceTypes, blockNum)
}

func (api *ArbTraceForwarderAPI) CallMany(ctx context.Context, calls json.RawMessage, blockNum json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_callMany", calls, blockNum)
}

func (api *ArbTraceForwarderAPI) ReplayBlockTransactions(ctx context.Context, blockNum json.RawMessage, traceTypes json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_replayBlockTransactions", blockNum, traceTypes)
}

func (api *ArbTraceForwarderAPI) ReplayTransaction(ctx context.Context, txHash json.RawMessage, traceTypes json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_replayTransaction", txHash, traceTypes)
}

func (api *ArbTraceForwarderAPI) Transaction(ctx context.Context, txHash json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_transaction", txHash)
}

func (api *ArbTraceForwarderAPI) Get(ctx context.Context, txHash json.RawMessage, path json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_get", txHash, path)
}

func (api *ArbTraceForwarderAPI) Block(ctx context.Context, blockNum json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_block", blockNum)
}

func (api *ArbTraceForwarderAPI) Filter(ctx context.Context, filter json.RawMessage) (*json.RawMessage, error) {
	return api.forward(ctx, "arbtrace_filter", filter)
}
