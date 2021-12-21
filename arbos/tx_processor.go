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
)

var arbAddress = common.HexToAddress("0xabc")
var networkAddress = common.HexToAddress("0x01")

type TxProcessor struct {
	msg          core.Message
	blockContext vm.BlockContext
	stateDB      vm.StateDB
	state        *ArbosState
	posterFee    *big.Int // set once in GasChargingHook to track L1 calldata costs
}

func NewTxProcessor(msg core.Message, evm *vm.EVM) *TxProcessor {
	arbosState := OpenArbosState(evm.StateDB)
	arbosState.SetLastTimestampSeen(evm.Context.Time.Uint64())
	return &TxProcessor{
		msg:          msg,
		blockContext: evm.Context,
		stateDB:      evm.StateDB,
		state:        arbosState,
		posterFee:    nil,
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

func (p *TxProcessor) InterceptMessage() *core.ExecutionResult {
	if p.msg.From() != arbAddress {
		return nil
	}
	// Message is deposit
	p.stateDB.AddBalance(*p.msg.To(), p.msg.Value())
	return &core.ExecutionResult{
		UsedGas:    0,
		Err:        nil,
		ReturnData: nil,
	}
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
		adjustedPrice = new(big.Int).Mul(adjustedPrice, big.NewInt(16))
		gasPrice = adjustedPrice
	}
	if gasPrice.Sign() > 0 {
		posterCostInL2Gas := new(big.Int).Div(posterCost, gasPrice)
		if !posterCostInL2Gas.IsUint64() {
			posterCostInL2Gas = new(big.Int).SetUint64(math.MaxUint64)
		}
		gasNeededToStartEVM = posterCostInL2Gas.Uint64()
	}

	p.posterFee = posterCost

	if *gasRemaining < gasNeededToStartEVM {
		return vm.ErrOutOfGas
	}
	*gasRemaining -= gasNeededToStartEVM
	return nil
}

func (p *TxProcessor) EndTxHook(gasLeft uint64, success bool) error {

	gasPrice := p.blockContext.BaseFee

	if gasLeft > p.msg.Gas() {
		panic("Tx somehow refunds gas after computation")
	}
	gasUsed := new(big.Int).SetUint64(p.msg.Gas() - gasLeft)

	totalCost := new(big.Int).Mul(gasPrice, gasUsed)
	computeCost := new(big.Int).Sub(totalCost, p.posterFee)
	if computeCost.Sign() < 0 {
		panic("total cost < poster cost")
	}

	p.stateDB.AddBalance(networkAddress, computeCost)
	p.stateDB.AddBalance(p.blockContext.Coinbase, p.posterFee)

	if p.msg.GasPrice().Sign() > 0 {
		// in tests, gasprice coud be 0
		p.state.notifyGasUsed(computeCost.Uint64())
	}
	return nil
}
