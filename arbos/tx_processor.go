//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math"
	"math/big"

	arbos_util "github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/util"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/retryables"

	"github.com/offchainlabs/arbstate/arbos/arbosState"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

var arbAddress = common.HexToAddress("0xabc")

// A TxProcessor is created and freed for every L2 transaction.
// It tracks state for ArbOS, allowing it infuence in Geth's tx processing.
// Public fields are accessible in precompiles.
type TxProcessor struct {
	msg       core.Message
	state     *arbosState.ArbosState
	PosterFee *big.Int // set once in GasChargingHook to track L1 calldata costs
	posterGas uint64
	Callers   []common.Address
	TopTxType *byte // set once in StartTxHook
	evm       *vm.EVM
}

func NewTxProcessor(evm *vm.EVM, msg core.Message) *TxProcessor {
	arbosState := arbosState.OpenSystemArbosState(evm.StateDB)
	arbosState.SetLastTimestampSeen(evm.Context.Time.Uint64())
	return &TxProcessor{
		msg:       msg,
		state:     arbosState,
		PosterFee: new(big.Int),
		posterGas: 0,
		Callers:   []common.Address{},
		TopTxType: nil,
		evm:       evm,
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
	coinbase := p.evm.Context.Coinbase
	if isAggregated(coinbase, p.msg.From()) {
		return &coinbase
	}
	return nil
}

func (p *TxProcessor) StartTxHook() (endTxNow bool, gasUsed uint64, err error, returnData []byte) {
	// This hook is called before gas charging and will end the state transition if endTxNow is set to true
	// Hence, we must charge for any l2 resources if endTxNow is returned true

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
		p.evm.StateDB.AddBalance(*p.msg.To(), p.msg.Value())
		return true, 0, nil, nil
	case *types.ArbitrumSubmitRetryableTx:
		statedb := p.evm.StateDB
		ticketId := underlyingTx.Hash()
		escrow := retryables.RetryableEscrowAddress(ticketId)

		statedb.AddBalance(tx.From, tx.DepositValue)

		err := arbos_util.TransferBalance(tx.From, escrow, tx.Value, statedb)
		if err != nil {
			return true, 0, err, nil
		}

		time := p.evm.Context.Time.Uint64()
		timeout := time + retryables.RetryableLifetimeSeconds

		// we charge for creating the retryable and reaping the next expired one on L1
		retryable, err := p.state.RetryableState().CreateRetryable(
			time,
			ticketId,
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
			log.Error("failed to emit TicketCreated event", "err", err)
		}

		balance := statedb.GetBalance(tx.From)
		basefee := p.evm.Context.BaseFee
		usergas := p.msg.Gas()
		gascost := util.BigMulByUint(basefee, usergas)

		if util.BigLessThan(balance, gascost) || usergas < params.TxGas {
			// user didn't have or provide enough gas to do an initial redeem
			return true, 0, nil, underlyingTx.Hash().Bytes()
		}

		if util.BigLessThan(tx.GasPrice, basefee) {
			// user's bid was too low
			return true, 0, nil, underlyingTx.Hash().Bytes()
		}

		// pay for the retryable's gas and update the pools
		networkFeeAccount, _ := p.state.NetworkFeeAccount()
		err = arbos_util.TransferBalance(tx.From, networkFeeAccount, gascost, statedb)
		if err != nil {
			// should be impossible because we just checked the tx.From balance
			panic(err)
		}
		p.state.L2PricingState().AddToGasPools(-util.SaturatingCast(usergas))

		// emit RedeemScheduled event
		retryTxInner, err := retryable.MakeTx(
			underlyingTx.ChainId(),
			0,
			basefee,
			usergas,
			ticketId,
			tx.FeeRefundAddr,
		)
		p.state.Restrict(err)

		_, err = retryable.IncrementNumTries()
		p.state.Restrict(err)

		err = EmitReedeemScheduledEvent(
			p.evm,
			usergas,
			retryTxInner.Nonce,
			ticketId,
			types.NewTx(retryTxInner).Hash(),
			tx.FeeRefundAddr,
		)
		if err != nil {
			log.Error("failed to emit RedeemScheduled event", "err", err)
		}

		return true, usergas, nil, underlyingTx.Hash().Bytes()
	case *types.ArbitrumRetryTx:

		// Transfer callvalue from escrow
		escrow := retryables.RetryableEscrowAddress(tx.TicketId)
		err = arbos_util.TransferBalance(escrow, tx.From, tx.Value, p.evm.StateDB)
		if err != nil {
			return true, 0, err, nil
		}

		// The redeemer has pre-paid for this tx's gas
		basefee := p.evm.Context.BaseFee
		p.evm.StateDB.AddBalance(tx.From, util.BigMulByUint(basefee, tx.Gas))
	}
	return false, 0, nil, nil
}

func (p *TxProcessor) GasChargingHook(gasRemaining *uint64) error {
	// Because a user pays a 1-dimensional gas price, we must re-express poster L1 calldata costs
	// as if the user was buying an equivalent amount of L2 compute gas. This hook determines what
	// that cost looks like, ensuring the user can pay and saving the result for later reference.

	var gasNeededToStartEVM uint64

	gasPrice := p.evm.Context.BaseFee
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
	gasPrice := p.evm.Context.BaseFee

	if underlyingTx != nil && underlyingTx.Type() == types.ArbitrumRetryTxType {
		inner, _ := underlyingTx.GetInner().(*types.ArbitrumRetryTx)
		refund := util.BigMulByUint(gasPrice, gasLeft)
		err := arbos_util.TransferBalance(inner.From, inner.RefundTo, refund, p.evm.StateDB)
		if err != nil {
			// should be impossible because geth has already credited the gas refund to inner.From
			panic(err)
		}
		if success {
			state := arbosState.OpenSystemArbosState(p.evm.StateDB) // we don't want to charge for this
			_, _ = state.RetryableState().DeleteRetryable(inner.TicketId)
		} else {
			// return the Callvalue to escrow
			escrow := retryables.RetryableEscrowAddress(inner.TicketId)
			err = arbos_util.TransferBalance(inner.From, escrow, inner.Value, p.evm.StateDB)
			if err != nil {
				// should be impossible because geth credited the inner.Value to inner.From before the transaction
				// and the transaction reverted
				panic(err)
			}
		}
		// we've already credited the network fee account and updated the gas pool
		p.state.L2PricingState().AddToGasPools(util.SaturatingCast(gasLeft))
		return
	}

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

	networkFeeAccount, _ := p.state.NetworkFeeAccount()
	p.evm.StateDB.AddBalance(networkFeeAccount, computeCost)
	p.evm.StateDB.AddBalance(p.evm.Context.Coinbase, p.PosterFee)

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
		p.state.L2PricingState().AddToGasPools(-int64(computeGas))
	}
}

func (p *TxProcessor) L1BlockNumber(blockCtx vm.BlockContext) (uint64, error) {
	state := arbosState.OpenSystemArbosState(p.evm.StateDB)
	return state.Blockhashes().NextBlockNumber()
}

func (p *TxProcessor) L1BlockHash(blockCtx vm.BlockContext, l1BlocKNumber uint64) (common.Hash, error) {
	state := arbosState.OpenSystemArbosState(p.evm.StateDB)
	return state.Blockhashes().BlockHash(l1BlocKNumber)
}
