//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/retryables"

	"github.com/offchainlabs/arbstate/arbos/arbosState"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/util"
)

var arbAddress = common.HexToAddress("0xabc")
var networkAddress = common.HexToAddress("0x01")

// A TxProcessor is created and freed for every L2 transaction.
// It tracks state for ArbOS, allowing it infuence in Geth's tx processing.
// Public fields are accessible in precompiles.
type TxProcessor struct {
	msg          core.Message
	blockContext vm.BlockContext
	stateDB      vm.StateDB
	state        *arbosState.ArbosState
	PosterFee    *big.Int // set once in GasChargingHook to track L1 calldata costs
	posterGas    uint64
	Callers      []common.Address
	TopTxType    *byte // set once in StartTxHook
	evm          *vm.EVM
}

func NewTxProcessor(evm *vm.EVM, msg core.Message) *TxProcessor {
	arbosState := arbosState.OpenSystemArbosState(evm.StateDB)
	arbosState.SetLastTimestampSeen(evm.Context.Time.Uint64())
	return &TxProcessor{
		msg:          msg,
		blockContext: evm.Context,
		stateDB:      evm.StateDB,
		state:        arbosState,
		PosterFee:    new(big.Int),
		posterGas:    0,
		Callers:      []common.Address{},
		TopTxType:    nil,
		evm:          evm,
	}
}

func (p *TxProcessor) PushCaller(addr common.Address) {
	p.Callers = append(p.Callers, addr)
}

func (p *TxProcessor) PopCaller() {
	p.Callers = p.Callers[:len(p.Callers)-1]
}

func isAggregated(l1Address, l2Address common.Address) bool {
	return true // TODO
}

func (p *TxProcessor) getAggregator() *common.Address {
	coinbase := p.blockContext.Coinbase
	if isAggregated(coinbase, p.msg.From()) {
		return &coinbase
	}
	return nil
}

func (p *TxProcessor) StartTxHook() (endTxNow bool, gasUsed uint64, err error, returnData []byte) {
	underlyingTx := p.msg.UnderlyingTransaction()
	if underlyingTx == nil {
		return false, 0, nil, nil
	}

	tipe := underlyingTx.Type()
	p.TopTxType = &tipe

	switch tx := underlyingTx.GetInner().(type) {
	case *types.ArbitrumDepositTx:
		if p.msg.From() != arbAddress {
			return false, 0, nil, nil
		}
		p.stateDB.AddBalance(*p.msg.To(), p.msg.Value())
		return true, 0, nil, nil
	case *types.ArbitrumSubmitRetryableTx:
		p.stateDB.AddBalance(tx.From, tx.DepositValue)

		time := p.blockContext.Time.Uint64()
		timeout := time + retryables.RetryableLifetimeSeconds

		retryable, err := p.state.RetryableState().CreateRetryable(
			time,
			underlyingTx.Hash(),
			timeout,
			tx.From,
			underlyingTx.To(),
			underlyingTx.Value(),
			tx.Beneficiary,
			tx.Data,
		)
		p.state.Restrict(err)

		err = EmitTicketCreatedEvent(p.evm, underlyingTx.Hash())
		if err != nil {
			log.Error("failed to emit RedeemScheduled event", "err", err)
		}

		if p.msg.Gas() > params.TxGas {
			// emit RedeemScheduled event
			retryTxInner, err := retryable.MakeTx(
				underlyingTx.ChainId(),
				0,
				p.blockContext.BaseFee,
				p.msg.Gas(),
				underlyingTx.Hash(),
				p.msg.From(),
			)
			p.state.Restrict(err)

			_, err = retryable.IncrementNumTries()
			p.state.Restrict(err)

			err = EmitReedeemScheduledEvent(
				p.evm,
				retryTxInner.Nonce,
				p.msg.Gas(),
				retryTxInner.Nonce,
				underlyingTx.Hash(),
				types.NewTx(retryTxInner).Hash(),
				p.msg.From(),
			)
			if err != nil {
				log.Error("failed to emit RedeemScheduled event", "err", err)
			}
		}

		return true, p.msg.Gas(), nil, underlyingTx.Hash().Bytes()
	case *types.ArbitrumRetryTx:
		p.state.AddToGasPools(util.SaturatingCast(tx.Gas))
		p.state.Restrict(err)
	}
	return false, 0, nil, nil
}

func (p *TxProcessor) GasChargingHook(gasRemaining *uint64) error {
	// Because a user pays a 1-dimensional gas price, we must re-express poster L1 calldata costs
	// as if the user was buying an equivalent amount of L2 compute gas. This hook determines what
	// that cost looks like, ensuring the user can pay and saving the result for later reference.

	var gasNeededToStartEVM uint64

	gasPrice := p.blockContext.BaseFee
	pricing := p.state.L1PricingState()
	posterCost, err := pricing.PosterDataCost(p.msg.From(), p.getAggregator(), p.msg.Data())
	p.state.Restrict(err)

	if p.msg.GasPrice().Sign() == 0 {
		// TODO: Review when doing eth_call's

		// Suggest the amount of gas needed for a given amount of ETH is higher in case of congestion.
		// This will help an eth_call user pad the total they'll pay in case the price rises a bit.
		// Note, reducing the poster cost will increase share the network fee gets, not reduce the total.
		adjustedPrice := util.BigMulByFrac(gasPrice, 15, 16)
		gasPrice = adjustedPrice
	}
	if gasPrice.Sign() > 0 {
		posterCostInL2Gas := new(big.Int).Div(posterCost, gasPrice) // the cost as if it were an amount of gas
		if !posterCostInL2Gas.IsUint64() {
			posterCostInL2Gas = new(big.Int).SetUint64(math.MaxUint64)
		}
		p.posterGas = posterCostInL2Gas.Uint64()
		p.PosterFee = new(big.Int).Mul(posterCostInL2Gas, gasPrice) // round down
		gasNeededToStartEVM = p.posterGas
	}

	if *gasRemaining < gasNeededToStartEVM {
		// the user couldn't pay for call data, so give up
		return vm.ErrOutOfGas
	}
	*gasRemaining -= gasNeededToStartEVM
	return nil
}

func (p *TxProcessor) NonrefundableGas() uint64 {
	// EVM-incentivized activity like freeing storage should only refund amounts paid to the network address,
	// which represents the overall burden to node operators. A poster's costs, then, should not be eligible
	// for this refund.
	return p.posterGas
}

func (p *TxProcessor) EndTxHook(gasLeft uint64, success bool) {

	underlyingTx := p.msg.UnderlyingTransaction()
	if success && underlyingTx != nil && underlyingTx.Type() == types.ArbitrumRetryTxType {
		state := arbosState.OpenSystemArbosState(p.stateDB) // we don't want to charge for this
		inner, _ := underlyingTx.GetInner().(*types.ArbitrumRetryTx)
		_, _ = state.RetryableState().DeleteRetryable(inner.TicketId)
	}

	gasPrice := p.blockContext.BaseFee

	if gasLeft > p.msg.Gas() {
		panic("Tx somehow refunds gas after computation")
	}
	gasUsed := new(big.Int).SetUint64(p.msg.Gas() - gasLeft)

	totalCost := new(big.Int).Mul(gasPrice, gasUsed)        // total cost = price of gas * gas burnt
	computeCost := new(big.Int).Sub(totalCost, p.PosterFee) // total cost = network's compute + poster's L1 costs
	if computeCost.Sign() < 0 {
		// Uh oh, there's a bug in our charging code.
		// Give all funds to the network account and continue.
		log.Error("total cost < poster cost", "gasUsed", gasUsed, "gasPrice", gasPrice, "posterFee", p.PosterFee)
		p.PosterFee = big.NewInt(0)
		computeCost = totalCost
	}

	p.stateDB.AddBalance(networkAddress, computeCost)
	p.stateDB.AddBalance(p.blockContext.Coinbase, p.PosterFee)

	if p.msg.GasPrice().Sign() > 0 { // in tests, gas price coud be 0
		// ArbOS's gas pool is meant to enforce the computational speed-limit.
		// We don't want to remove from the pool the poster's L1 costs (as expressed in L2 gas in this func)
		// Hence, we deduct the previously saved poster L2-gas-equivalent to reveal the compute-only gas

		if gasUsed.Uint64() < p.posterGas {
			log.Error("total gas used < poster gas component", "gasUsed", gasUsed, "posterGas", p.posterGas)
		}
		computeGas := gasUsed.Uint64() - p.posterGas
		if computeGas > math.MaxInt64 {
			computeGas = math.MaxInt64
		}
		p.state.AddToGasPools(-int64(computeGas))
	}
}

func (p *TxProcessor) L1BlockNumber(blockCtx vm.BlockContext) (uint64, error) {
	state := arbosState.OpenSystemArbosState(p.stateDB)
	return state.Blockhashes().NextBlockNumber()
}

func (p *TxProcessor) L1BlockHash(blockCtx vm.BlockContext, l1BlocKNumber uint64) (common.Hash, error) {
	state := arbosState.OpenSystemArbosState(p.stateDB)
	return state.Blockhashes().BlockHash(l1BlocKNumber)
}
