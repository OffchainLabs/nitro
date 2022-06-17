// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/arbos/l1pricing"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/retryables"

	"github.com/offchainlabs/nitro/arbos/arbosState"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	glog "github.com/ethereum/go-ethereum/log"
)

var arbosAddress = types.ArbosAddress

// A TxProcessor is created and freed for every L2 transaction.
// It tracks state for ArbOS, allowing it infuence in Geth's tx processing.
// Public fields are accessible in precompiles.
type TxProcessor struct {
	msg              core.Message
	state            *arbosState.ArbosState
	PosterFee        *big.Int // set once in GasChargingHook to track L1 calldata costs
	posterGas        uint64
	computeHoldGas   uint64 // amount of gas temporarily held to prevent compute from exceeding the gas limit
	Callers          []common.Address
	TopTxType        *byte // set once in StartTxHook
	evm              *vm.EVM
	CurrentRetryable *common.Hash
}

func NewTxProcessor(evm *vm.EVM, msg core.Message) *TxProcessor {
	tracingInfo := util.NewTracingInfo(evm, msg.From(), arbosAddress, util.TracingBeforeEVM)
	arbosState := arbosState.OpenSystemArbosStateOrPanic(evm.StateDB, tracingInfo, false)
	return &TxProcessor{
		msg:              msg,
		state:            arbosState,
		PosterFee:        new(big.Int),
		posterGas:        0,
		Callers:          []common.Address{},
		TopTxType:        nil,
		evm:              evm,
		CurrentRetryable: nil,
	}
}

func (p *TxProcessor) PushCaller(addr common.Address) {
	p.Callers = append(p.Callers, addr)
}

func (p *TxProcessor) PopCaller() {
	p.Callers = p.Callers[:len(p.Callers)-1]
}

func (p *TxProcessor) StartTxHook() (endTxNow bool, gasUsed uint64, err error, returnData []byte) {
	// This hook is called before gas charging and will end the state transition if endTxNow is set to true
	// Hence, we must charge for any l2 resources if endTxNow is returned true

	underlyingTx := p.msg.UnderlyingTransaction()
	if underlyingTx == nil {
		return false, 0, nil, nil
	}

	var tracingInfo *util.TracingInfo
	from := p.msg.From()
	tipe := underlyingTx.Type()
	p.TopTxType = &tipe
	evm := p.evm

	startTracer := func() func() {
		if !evm.Config.Debug {
			return func() {}
		}
		evm.IncrementDepth() // fake a call
		tracer := evm.Config.Tracer
		start := time.Now()
		tracer.CaptureStart(evm, from, *p.msg.To(), false, p.msg.Data(), p.msg.Gas(), p.msg.Value())

		tracingInfo = util.NewTracingInfo(evm, from, *p.msg.To(), util.TracingDuringEVM)
		p.state = arbosState.OpenSystemArbosStateOrPanic(evm.StateDB, tracingInfo, false)

		return func() {
			tracer.CaptureEnd(nil, p.state.Burner.Burned(), time.Since(start), nil)
			evm.DecrementDepth() // fake the return to the first faked call

			tracingInfo = util.NewTracingInfo(evm, from, *p.msg.To(), util.TracingAfterEVM)
			p.state = arbosState.OpenSystemArbosStateOrPanic(evm.StateDB, tracingInfo, false)
		}
	}

	switch tx := underlyingTx.GetInner().(type) {
	case *types.ArbitrumDepositTx:
		defer (startTracer())()
		if p.msg.From() != arbosAddress {
			return false, 0, errors.New("deposit not from arbAddress"), nil
		}
		util.MintBalance(p.msg.To(), p.msg.Value(), evm, util.TracingDuringEVM, "deposit")
		return true, 0, nil, nil
	case *types.ArbitrumInternalTx:
		defer (startTracer())()
		if p.msg.From() != arbosAddress {
			return false, 0, errors.New("internal tx not from arbAddress"), nil
		}
		ApplyInternalTxUpdate(tx, p.state, evm)
		return true, 0, nil, nil
	case *types.ArbitrumSubmitRetryableTx:
		defer (startTracer())()
		statedb := evm.StateDB
		ticketId := underlyingTx.Hash()
		escrow := retryables.RetryableEscrowAddress(ticketId)
		networkFeeAccount, _ := p.state.NetworkFeeAccount()
		from := tx.From
		scenario := util.TracingDuringEVM

		// mint funds with the deposit, then charge fees later
		util.MintBalance(&from, tx.DepositValue, evm, scenario, "deposit")

		submissionFee := retryables.RetryableSubmissionFee(len(tx.RetryData), tx.L1BaseFee)
		excessDeposit := arbmath.BigSub(tx.MaxSubmissionFee, submissionFee)
		if excessDeposit.Sign() < 0 {
			return true, 0, errors.New("max submission fee is less than the actual submission fee"), nil
		}

		transfer := func(from, to *common.Address, amount *big.Int) error {
			return util.TransferBalance(from, to, amount, evm, scenario, "during evm execution")
		}

		// move balance to the relevant parties
		if err := transfer(&from, &networkFeeAccount, submissionFee); err != nil {
			return true, 0, err, nil
		}
		if err := transfer(&from, &tx.FeeRefundAddr, excessDeposit); err != nil {
			return true, 0, err, nil
		}
		if err := transfer(&tx.From, &escrow, tx.Value); err != nil {
			return true, 0, err, nil
		}

		time := evm.Context.Time.Uint64()
		timeout := time + retryables.RetryableLifetimeSeconds

		// we charge for creating the retryable and reaping the next expired one on L1
		retryable, err := p.state.RetryableState().CreateRetryable(
			ticketId,
			timeout,
			tx.From,
			tx.RetryTo,
			underlyingTx.Value(),
			tx.Beneficiary,
			tx.RetryData,
		)
		p.state.Restrict(err)

		err = EmitTicketCreatedEvent(evm, underlyingTx.Hash())
		if err != nil {
			glog.Error("failed to emit TicketCreated event", "err", err)
		}

		balance := statedb.GetBalance(tx.From)
		basefee := evm.Context.BaseFee
		usergas := p.msg.Gas()
		gascost := arbmath.BigMulByUint(basefee, usergas)

		if arbmath.BigLessThan(balance, gascost) || usergas < params.TxGas {
			// user didn't have or provide enough gas to do an initial redeem
			return true, 0, nil, underlyingTx.Hash().Bytes()
		}

		if arbmath.BigLessThan(tx.GasFeeCap, basefee) && p.msg.RunMode() != types.MessageGasEstimationMode {
			// user's bid was too low
			return true, 0, nil, underlyingTx.Hash().Bytes()
		}

		// pay for the retryable's gas and update the pools
		if transfer(&tx.From, &networkFeeAccount, gascost) != nil {
			// should be impossible because we just checked the tx.From balance
			panic(err)
		}

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
			evm,
			usergas,
			retryTxInner.Nonce,
			ticketId,
			types.NewTx(retryTxInner).Hash(),
			tx.FeeRefundAddr,
		)
		if err != nil {
			glog.Error("failed to emit RedeemScheduled event", "err", err)
		}

		if evm.Config.Debug {
			redeem, err := util.PackArbRetryableTxRedeem(ticketId)
			if err == nil {
				tracingInfo.MockCall(redeem, usergas, from, types.ArbRetryableTxAddress, tx.Value)
			} else {
				glog.Error("failed to abi-encode auto-redeem", "err", err)
			}
		}

		return true, usergas, nil, underlyingTx.Hash().Bytes()
	case *types.ArbitrumRetryTx:

		// Transfer callvalue from escrow
		escrow := retryables.RetryableEscrowAddress(tx.TicketId)
		scenario := util.TracingBeforeEVM
		if util.TransferBalance(&escrow, &tx.From, tx.Value, evm, scenario, "escrow") != nil {
			return true, 0, err, nil
		}

		// The redeemer has pre-paid for this tx's gas
		prepaid := arbmath.BigMulByUint(evm.Context.BaseFee, tx.Gas)
		util.MintBalance(&tx.From, prepaid, evm, scenario, "prepaid")
		ticketId := tx.TicketId
		p.CurrentRetryable = &ticketId
	}
	return false, 0, nil, nil
}

func (p *TxProcessor) GasChargingHook(gasRemaining *uint64) error {
	// Because a user pays a 1-dimensional gas price, we must re-express poster L1 calldata costs
	// as if the user was buying an equivalent amount of L2 compute gas. This hook determines what
	// that cost looks like, ensuring the user can pay and saving the result for later reference.

	var gasNeededToStartEVM uint64
	gasPrice := p.evm.Context.BaseFee

	var poster common.Address
	if p.msg.RunMode() == types.MessageGasEstimationMode {
		poster = l1pricing.BatchPosterAddress
	} else {
		poster = p.evm.Context.Coinbase
	}
	posterCost, calldataUnits := p.state.L1PricingState().PosterDataCost(p.msg, poster)
	if calldataUnits > 0 {
		if err := p.state.L1PricingState().AddToUnitsSinceUpdate(calldataUnits); err != nil {
			return err
		}
	}

	if p.msg.RunMode() == types.MessageGasEstimationMode {
		// Suggest the amount of gas needed for a given amount of ETH is higher in case of congestion.
		// This will help the user pad the total they'll pay in case the price rises a bit.
		// Note, reducing the poster cost will increase share the network fee gets, not reduce the total.

		minGasPrice, _ := p.state.L2PricingState().MinBaseFeeWei()

		adjustedPrice := arbmath.BigMulByFrac(gasPrice, 7, 8) // assume congestion
		if arbmath.BigLessThan(adjustedPrice, minGasPrice) {
			adjustedPrice = minGasPrice
		}
		gasPrice = adjustedPrice

		// Pad the L1 cost by 10% in case the L1 gas price rises
		posterCost = arbmath.BigMulByFrac(posterCost, 110, 100)
	}

	if gasPrice.Sign() > 0 {
		posterCostInL2Gas := arbmath.BigDiv(posterCost, gasPrice) // the cost as if it were an amount of gas
		if !posterCostInL2Gas.IsUint64() {
			posterCostInL2Gas = arbmath.UintToBig(math.MaxUint64)
		}
		p.posterGas = posterCostInL2Gas.Uint64()
		p.PosterFee = arbmath.BigMul(posterCostInL2Gas, gasPrice) // round down
		gasNeededToStartEVM = p.posterGas
	}

	if *gasRemaining < gasNeededToStartEVM {
		// the user couldn't pay for call data, so give up
		return core.ErrIntrinsicGas
	}
	*gasRemaining -= gasNeededToStartEVM

	if p.msg.RunMode() != types.MessageEthcallMode {
		// If this is a real tx, limit the amount of computed based on the gas pool.
		// We do this by charging extra gas, and then refunding it later.
		gasAvailable, _ := p.state.L2PricingState().PerBlockGasLimit()
		if *gasRemaining > gasAvailable {
			p.computeHoldGas = *gasRemaining - gasAvailable
			*gasRemaining = gasAvailable
		}
	}
	return nil
}

func (p *TxProcessor) NonrefundableGas() uint64 {
	// EVM-incentivized activity like freeing storage should only refund amounts paid to the network address,
	// which represents the overall burden to node operators. A poster's costs, then, should not be eligible
	// for this refund.
	return p.posterGas
}

func (p *TxProcessor) ForceRefundGas() uint64 {
	return p.computeHoldGas
}

func (p *TxProcessor) EndTxHook(gasLeft uint64, success bool) {

	underlyingTx := p.msg.UnderlyingTransaction()
	gasPrice := p.evm.Context.BaseFee
	networkFeeAccount, _ := p.state.NetworkFeeAccount()
	scenario := util.TracingAfterEVM

	if gasLeft > p.msg.Gas() {
		panic("Tx somehow refunds gas after computation")
	}
	gasUsed := p.msg.Gas() - gasLeft

	if underlyingTx != nil && underlyingTx.Type() == types.ArbitrumRetryTxType {
		inner, _ := underlyingTx.GetInner().(*types.ArbitrumRetryTx)
		refund := arbmath.BigMulByUint(gasPrice, gasLeft)

		// undo Geth's refund to the From address
		err := util.TransferBalance(&inner.From, nil, refund, p.evm, scenario, "undoRefund")
		if err != nil {
			log.Error("Uh oh, Geth didn't refund the user", inner.From, refund)
		}

		// refund the RefundTo by taking fees back from the network address
		err = util.TransferBalance(&networkFeeAccount, &inner.RefundTo, refund, p.evm, scenario, "refund")
		if err != nil {
			// Normally the network fee address should be holding the gas funds.
			// However, in theory, they could've been transfered out during the redeem attempt.
			// If the network fee address doesn't have the necessary balance, log an error and don't give a refund.
			log.Error("network fee address doesn't have enough funds to give user refund", "err", err)
		}
		if success {
			// we don't want to charge for this
			tracingInfo := util.NewTracingInfo(p.evm, arbosAddress, p.msg.From(), scenario)
			state := arbosState.OpenSystemArbosStateOrPanic(p.evm.StateDB, tracingInfo, false)
			_, _ = state.RetryableState().DeleteRetryable(inner.TicketId, p.evm, scenario)
		} else {
			// return the Callvalue to escrow
			escrow := retryables.RetryableEscrowAddress(inner.TicketId)
			err := util.TransferBalance(&inner.From, &escrow, inner.Value, p.evm, scenario, "escrow")
			if err != nil {
				// should be impossible because geth credited the inner.Value to inner.From before the transaction
				// and the transaction reverted
				panic(err)
			}
		}
		// we've already credited the network fee account, but we didn't charge the gas pool yet
		p.state.Restrict(p.state.L2PricingState().AddToGasPool(-arbmath.SaturatingCast(gasUsed)))
		return
	}

	totalCost := arbmath.BigMul(gasPrice, arbmath.UintToBig(gasUsed)) // total cost = price of gas * gas burnt
	computeCost := arbmath.BigSub(totalCost, p.PosterFee)             // total cost = network's compute + poster's L1 costs
	if computeCost.Sign() < 0 {
		// Uh oh, there's a bug in our charging code.
		// Give all funds to the network account and continue.

		log.Error("total cost < poster cost", "gasUsed", gasUsed, "gasPrice", gasPrice, "posterFee", p.PosterFee)
		p.PosterFee = big.NewInt(0)
		computeCost = totalCost
	}

	purpose := "feeCollection"
	util.MintBalance(&networkFeeAccount, computeCost, p.evm, scenario, purpose)
	util.MintBalance(&p.evm.Context.Coinbase, p.PosterFee, p.evm, scenario, purpose)

	if p.msg.GasPrice().Sign() > 0 { // in tests, gas price coud be 0
		// ArbOS's gas pool is meant to enforce the computational speed-limit.
		// We don't want to remove from the pool the poster's L1 costs (as expressed in L2 gas in this func)
		// Hence, we deduct the previously saved poster L2-gas-equivalent to reveal the compute-only gas

		var computeGas uint64
		if gasUsed > p.posterGas {
			// Don't include posterGas in computeGas as it doesn't represent processing time.
			computeGas = gasUsed - p.posterGas
		} else {
			// Somehow, the core message transition succeeded, but we didn't burn the posterGas.
			// An invariant was violated. To be safe, subtract the entire gas used from the gas pool.
			log.Error("total gas used < poster gas component", "gasUsed", gasUsed, "posterGas", p.posterGas)
			computeGas = gasUsed
		}
		p.state.Restrict(p.state.L2PricingState().AddToGasPool(-arbmath.SaturatingCast(computeGas)))
	}
}

func (p *TxProcessor) ScheduledTxes() types.Transactions {
	scheduled := types.Transactions{}
	time := p.evm.Context.Time.Uint64()
	basefee := p.evm.Context.BaseFee
	chainID := p.evm.ChainConfig().ChainID

	logs := p.evm.StateDB.GetCurrentTxLogs()
	for _, log := range logs {
		if log.Address != ArbRetryableTxAddress || log.Topics[0] != RedeemScheduledEventID {
			continue
		}
		event := &precompilesgen.ArbRetryableTxRedeemScheduled{}
		err := util.ParseRedeemScheduledLog(event, log)
		if err != nil {
			glog.Error("Failed to parse RedeemScheduled log", "err", err)
			continue
		}
		retryable, err := p.state.RetryableState().OpenRetryable(event.TicketId, time)
		if err != nil || retryable == nil {
			continue
		}
		redeem, _ := retryable.MakeTx(
			chainID,
			event.SequenceNum,
			basefee,
			event.DonatedGas,
			event.TicketId,
			event.GasDonor,
		)
		scheduled = append(scheduled, types.NewTx(redeem))
	}
	return scheduled
}

func (p *TxProcessor) L1BlockNumber(blockCtx vm.BlockContext) (uint64, error) {
	tracingInfo := util.NewTracingInfo(p.evm, p.msg.From(), arbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenSystemArbosState(p.evm.StateDB, tracingInfo, false)
	if err != nil {
		return 0, err
	}
	return state.Blockhashes().NextBlockNumber()
}

func (p *TxProcessor) L1BlockHash(blockCtx vm.BlockContext, l1BlockNumber uint64) (common.Hash, error) {
	tracingInfo := util.NewTracingInfo(p.evm, p.msg.From(), arbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenSystemArbosState(p.evm.StateDB, tracingInfo, false)
	if err != nil {
		return common.Hash{}, err
	}
	return state.Blockhashes().BlockHash(l1BlockNumber)
}

func (p *TxProcessor) FillReceiptInfo(receipt *types.Receipt) {
	receipt.GasUsedForL1 = p.posterGas
}
