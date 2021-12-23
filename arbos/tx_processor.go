//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
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
	state        *ArbosState
	PosterFee    *big.Int // set once in GasChargingHook to track L1 calldata costs
	posterGas    uint64
}

func NewTxProcessor(evm *vm.EVM, msg core.Message) *TxProcessor {
	arbosState := OpenArbosState(evm.StateDB)
	arbosState.SetLastTimestampSeen(evm.Context.Time.Uint64())
	return &TxProcessor{
		msg:          msg,
		blockContext: evm.Context,
		stateDB:      evm.StateDB,
		state:        arbosState,
		PosterFee:    new(big.Int),
		posterGas:    0,
	}
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

func (p *TxProcessor) InterceptMessage() bool {
	if p.msg.From() != arbAddress {
		return false
	}
	// Message is deposit
	p.stateDB.AddBalance(*p.msg.To(), p.msg.Value())
	return true
}

func (p *TxProcessor) GasChargingHook(gasRemaining *uint64) error {

	var gasNeededToStartEVM uint64

	gasPrice := p.blockContext.BaseFee
	pricing := p.state.L1PricingState()
	posterCost := pricing.PosterDataCost(p.msg.From(), p.getAggregator(), p.msg.Data())

	if p.msg.GasPrice().Sign() == 0 {
		// TODO: Review when doing eth_call's
		// suggest the amount of gas needed for a given amount of ETH is higher in case of congestion
		adjustedPrice := new(big.Int).Mul(gasPrice, big.NewInt(15))
		adjustedPrice = new(big.Int).Div(adjustedPrice, big.NewInt(16))
		gasPrice = adjustedPrice
	}
	if gasPrice.Sign() > 0 {
		posterCostInL2Gas := new(big.Int).Div(posterCost, gasPrice)
		if !posterCostInL2Gas.IsUint64() {
			posterCostInL2Gas = new(big.Int).SetUint64(math.MaxUint64)
		}
		p.posterGas = posterCostInL2Gas.Uint64()
		p.PosterFee = posterCost
		gasNeededToStartEVM = p.posterGas
	}

	if *gasRemaining < gasNeededToStartEVM {
		return vm.ErrOutOfGas
	}
	*gasRemaining -= gasNeededToStartEVM
	return nil
}

func (p *TxProcessor) NonrefundableGas() uint64 {
	return p.posterGas
}

func (p *TxProcessor) EndTxHook(gasLeft uint64, success bool) error {

	gasPrice := p.blockContext.BaseFee

	if gasLeft > p.msg.Gas() {
		panic("Tx somehow refunds gas after computation")
	}
	gasUsed := new(big.Int).SetUint64(p.msg.Gas() - gasLeft)

	totalCost := new(big.Int).Mul(gasPrice, gasUsed)
	computeCost := new(big.Int).Sub(totalCost, p.PosterFee)
	if computeCost.Sign() < 0 {
		log.Error("total cost < poster cost", "gasUsed", gasUsed, "gasPrice", gasPrice, "posterFee", p.PosterFee)
		p.PosterFee = totalCost
		computeCost = new(big.Int)
	}

	p.stateDB.AddBalance(networkAddress, computeCost)
	p.stateDB.AddBalance(p.blockContext.Coinbase, p.PosterFee)

	if p.msg.GasPrice().Sign() > 0 {
		// in tests, gasprice coud be 0
		p.state.notifyGasUsed(computeCost.Uint64())
	}
	return nil
}
