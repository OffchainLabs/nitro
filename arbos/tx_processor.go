package arbos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

var networkFeeCollector common.Address

type TxProcessor struct {
	msg core.Message
	blockContext vm.BlockContext
	stateDB vm.StateDB
	state *ArbosState
}

func NewTxProcessor(msg core.Message, evm *vm.EVM) *TxProcessor {
	return &TxProcessor{
		msg: msg,
		blockContext: evm.Context,
		stateDB:      evm.StateDB,
		state: OpenArbosState(evm.StateDB, evm.Context.Time.Uint64()),
	}
}

func isAggregated(l1Address, l2Address common.Address) bool {
	return true
}

func (p *TxProcessor) getAggregator() *common.Address {
	coinbase := p.blockContext.Coinbase
	if isAggregated(coinbase, p.msg.From()) {
		return &coinbase
	}
	return nil
}

func (p *TxProcessor) getExtraGasChargeWei() *big.Int { // returns wei to charge
	//TODO
	return big.NewInt(0)
}

func (p *TxProcessor) getL1GasCharge() uint64 {
	extraGasChargeWei := p.getExtraGasChargeWei()
	gasPrice := p.msg.GasPrice()
	if gasPrice.Cmp(big.NewInt(0)) == 0 {
		return 0
	}
	l1ChargesBig := new(big.Int).Div(extraGasChargeWei, gasPrice)
	if !l1ChargesBig.IsUint64() {
		return math.MaxUint64
	}
	return l1ChargesBig.Uint64()
}

func (p *TxProcessor) InterceptMessage() (*core.ExecutionResult, error) {
	if p.msg.From() != arbAddress {
		return nil, nil
	}
	// Message is deposit
	p.stateDB.AddBalance(*p.msg.To(), p.msg.Value())
	return &core.ExecutionResult{
		UsedGas:    0,
		Err:        nil,
		ReturnData: nil,
	}, nil
}

func (p *TxProcessor) ExtraGasChargingHook(gasRemaining *uint64, gasPool *core.GasPool) error {
	l1Charges := p.getL1GasCharge()
	if *gasRemaining < l1Charges {
		return vm.ErrOutOfGas
	}
	*gasRemaining -= l1Charges
	*gasPool = *gasPool.AddGas(l1Charges)
	return nil
}

func (p *TxProcessor) EndTxHook(gasLeft uint64, gasPool *core.GasPool, success bool) error {
	gasUsed := new(big.Int).SetUint64(p.msg.Gas() - gasLeft)
	totalPaid := new(big.Int).Mul(gasUsed, p.msg.GasPrice())
	l1ChargeWei := p.getExtraGasChargeWei()
	l2ChargeWei := new(big.Int).Sub(totalPaid, l1ChargeWei)
	p.stateDB.SubBalance(p.blockContext.Coinbase, l2ChargeWei)
	p.stateDB.AddBalance(networkFeeCollector, l2ChargeWei)
	p.state.notifyGasUsed(new(big.Int).Div(l2ChargeWei, p.msg.GasPrice()).Uint64())
	return nil
}
