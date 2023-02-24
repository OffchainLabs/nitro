// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"errors"
	"fmt"
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

const GasEstimationL1PricePadding arbmath.Bips = 11000 // pad estimates by 10%

// A TxProcessor is created and freed for every L2 transaction.
// It tracks state for ArbOS, allowing it infuence in Geth's tx processing.
// Public fields are accessible in precompiles.
type TxProcessor struct {
	msg              core.Message
	state            *arbosState.ArbosState
	PosterFee        *big.Int // set once in GasChargingHook to track L1 calldata costs
	posterGas        uint64
	computeHoldGas   uint64 // amount of gas temporarily held to prevent compute from exceeding the gas limit
	delayedInbox     bool   // whether this tx was submitted through the delayed inbox
	Callers          []common.Address
	TopTxType        *byte // set once in StartTxHook
	evm              *vm.EVM
	CurrentRetryable *common.Hash
	CurrentRefundTo  *common.Address

	// Caches for the latest L1 block number and hash,
	// for the NUMBER and BLOCKHASH opcodes.
	cachedL1BlockNumber *uint64
	cachedL1BlockHashes map[uint64]common.Hash
}

func NewTxProcessor(evm *vm.EVM, msg core.Message) *TxProcessor {
	tracingInfo := util.NewTracingInfo(evm, msg.From(), arbosAddress, util.TracingBeforeEVM)
	arbosState := arbosState.OpenSystemArbosStateOrPanic(evm.StateDB, tracingInfo, false)
	return &TxProcessor{
		msg:                 msg,
		state:               arbosState,
		PosterFee:           new(big.Int),
		posterGas:           0,
		delayedInbox:        evm.Context.Coinbase != l1pricing.BatchPosterAddress,
		Callers:             []common.Address{},
		TopTxType:           nil,
		evm:                 evm,
		CurrentRetryable:    nil,
		CurrentRefundTo:     nil,
		cachedL1BlockNumber: nil,
		cachedL1BlockHashes: make(map[uint64]common.Hash),
	}
}

func (p *TxProcessor) PushCaller(addr common.Address) {
	p.Callers = append(p.Callers, addr)
}

func (p *TxProcessor) PopCaller() {
	p.Callers = p.Callers[:len(p.Callers)-1]
}

// Attempts to subtract up to `take` from `pool` without going negative.
// Returns the amount subtracted from `pool`.
func takeFunds(pool *big.Int, take *big.Int) *big.Int {
	if take.Sign() < 0 {
		panic("Attempted to take a negative amount of funds")
	}
	if arbmath.BigLessThan(pool, take) {
		oldPool := new(big.Int).Set(pool)
		pool.Set(common.Big0)
		return oldPool
	} else {
		pool.Sub(pool, take)
		return new(big.Int).Set(take)
	}
}

func (p *TxProcessor) StartTxHook() (endTxNow bool, gasUsed uint64, err error, returnData []byte) {
	// This hook is called before gas charging and will end the state transition if endTxNow is set to true
	// Hence, we must charge for any l2 resources if endTxNow is returned true

	underlyingTx := p.msg.UnderlyingTransaction()
	if underlyingTx == nil {
		return false, 0, nil, nil
	}

	var tracingInfo *util.TracingInfo
	tipe := underlyingTx.Type()
	p.TopTxType = &tipe
	evm := p.evm

	startTracer := func() func() {
		if !evm.Config.Debug {
			return func() {}
		}
		evm.IncrementDepth() // fake a call
		tracer := evm.Config.Tracer
		from := p.msg.From()
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
		from := p.msg.From()
		to := p.msg.To()
		value := p.msg.Value()
		if to == nil {
			return true, 0, errors.New("eth deposit has no To address"), nil
		}
		util.MintBalance(&from, value, evm, util.TracingBeforeEVM, "deposit")
		defer (startTracer())()
		// We intentionally use the variant here that doesn't do tracing,
		// because this transfer is represented as the outer eth transaction.
		// This transfer is necessary because we don't actually invoke the EVM.
		core.Transfer(evm.StateDB, from, *to, value)
		return true, 0, nil, nil
	case *types.ArbitrumInternalTx:
		defer (startTracer())()
		if p.msg.From() != arbosAddress {
			return false, 0, errors.New("internal tx not from arbAddress"), nil
		}
		err = ApplyInternalTxUpdate(tx, p.state, evm)
		return true, 0, err, nil
	case *types.ArbitrumSubmitRetryableTx:
		defer (startTracer())()
		statedb := evm.StateDB
		ticketId := underlyingTx.Hash()
		escrow := retryables.RetryableEscrowAddress(ticketId)
		networkFeeAccount, _ := p.state.NetworkFeeAccount()
		from := tx.From
		scenario := util.TracingDuringEVM

		// mint funds with the deposit, then charge fees later
		availableRefund := new(big.Int).Set(tx.DepositValue)
		takeFunds(availableRefund, tx.RetryValue)
		util.MintBalance(&tx.From, tx.DepositValue, evm, scenario, "deposit")

		transfer := func(from, to *common.Address, amount *big.Int) error {
			return util.TransferBalance(from, to, amount, evm, scenario, "during evm execution")
		}

		// check that the user has enough balance to pay for the max submission fee
		balanceAfterMint := evm.StateDB.GetBalance(tx.From)
		if balanceAfterMint.Cmp(tx.MaxSubmissionFee) < 0 {
			err := fmt.Errorf(
				"insufficient funds for max submission fee: address %v have %v want %v",
				tx.From, balanceAfterMint, tx.MaxSubmissionFee,
			)
			return true, 0, err, nil
		}

		submissionFee := retryables.RetryableSubmissionFee(len(tx.RetryData), tx.L1BaseFee)
		if arbmath.BigLessThan(tx.MaxSubmissionFee, submissionFee) {
			// should be impossible as this is checked at L1
			err := fmt.Errorf(
				"max submission fee %v is less than the actual submission fee %v",
				tx.MaxSubmissionFee, submissionFee,
			)
			return true, 0, err, nil
		}

		// collect the submission fee
		if err := transfer(&tx.From, &networkFeeAccount, submissionFee); err != nil {
			// should be impossible as we just checked that they have enough balance for the max submission fee,
			// and we also checked that the max submission fee is at least the actual submission fee
			glog.Error("failed to transfer submissionFee", "err", err)
			return true, 0, err, nil
		}
		withheldSubmissionFee := takeFunds(availableRefund, submissionFee)

		// refund excess submission fee
		submissionFeeRefund := takeFunds(availableRefund, arbmath.BigSub(tx.MaxSubmissionFee, submissionFee))
		if err := transfer(&tx.From, &tx.FeeRefundAddr, submissionFeeRefund); err != nil {
			// should never happen as from's balance should be at least availableRefund at this point
			glog.Error("failed to transfer submissionFeeRefund", "err", err)
		}

		// move the callvalue into escrow
		if callValueErr := transfer(&tx.From, &escrow, tx.RetryValue); callValueErr != nil {
			// The sender doesn't have enough balance to pay for the retryable's callvalue.
			// Since we can't create the retryable, we should refund the submission fee.
			// First, we give the submission fee back to the transaction sender:
			if err := transfer(&networkFeeAccount, &tx.From, submissionFee); err != nil {
				glog.Error("failed to refund submissionFee", "err", err)
			}
			// Then, as limited by availableRefund, we attempt to move the refund to the fee refund address.
			// If the deposit value was lower than the submission fee, only some (or none) of the submission fee may be moved.
			// In that case, any amount up to the deposit value will be refunded to the fee refund address,
			// with the rest remaining in the transaction sender's address (as that's where the funds were pulled from).
			if err := transfer(&tx.From, &tx.FeeRefundAddr, withheldSubmissionFee); err != nil {
				glog.Error("failed to refund withheldSubmissionFee", "err", err)
			}
			return true, 0, callValueErr, nil
		}

		time := evm.Context.Time.Uint64()
		timeout := time + retryables.RetryableLifetimeSeconds

		// we charge for creating the retryable and reaping the next expired one on L1
		retryable, err := p.state.RetryableState().CreateRetryable(
			ticketId,
			timeout,
			tx.From,
			tx.RetryTo,
			tx.RetryValue,
			tx.Beneficiary,
			tx.RetryData,
		)
		p.state.Restrict(err)

		err = EmitTicketCreatedEvent(evm, ticketId)
		if err != nil {
			glog.Error("failed to emit TicketCreated event", "err", err)
		}

		balance := statedb.GetBalance(tx.From)
		basefee := evm.Context.BaseFee
		usergas := p.msg.Gas()

		maxGasCost := arbmath.BigMulByUint(tx.GasFeeCap, usergas)
		maxFeePerGasTooLow := arbmath.BigLessThan(tx.GasFeeCap, basefee)
		if p.msg.RunMode() == types.MessageGasEstimationMode && tx.GasFeeCap.BitLen() == 0 {
			// In gas estimation mode, we permit a zero gas fee cap.
			// This matches behavior with normal tx gas estimation.
			maxFeePerGasTooLow = false
		}
		if arbmath.BigLessThan(balance, maxGasCost) || usergas < params.TxGas || maxFeePerGasTooLow {
			// User either specified too low of a gas fee cap, didn't have enough balance to pay for gas,
			// or the specified gas limit is below the minimum transaction gas cost.
			// Either way, attempt to refund the gas costs, since we're not doing the auto-redeem.
			gasCostRefund := takeFunds(availableRefund, maxGasCost)
			if err := transfer(&tx.From, &tx.FeeRefundAddr, gasCostRefund); err != nil {
				// should never happen as from's balance should be at least availableRefund at this point
				glog.Error("failed to transfer gasCostRefund", "err", err)
			}
			return true, 0, nil, ticketId.Bytes()
		}

		// pay for the retryable's gas and update the pools
		gascost := arbmath.BigMulByUint(basefee, usergas)
		if err := transfer(&tx.From, &networkFeeAccount, gascost); err != nil {
			// should be impossible because we just checked the tx.From balance
			glog.Error("failed to transfer gas cost to network fee account", "err", err)
			return true, 0, nil, ticketId.Bytes()
		}

		withheldGasFunds := takeFunds(availableRefund, gascost) // gascost is conceptually charged before the gas price refund
		gasPriceRefund := arbmath.BigMulByUint(arbmath.BigSub(tx.GasFeeCap, basefee), tx.Gas)
		if gasPriceRefund.Sign() < 0 {
			// This should only be possible during gas estimation mode
			gasPriceRefund.SetInt64(0)
		}
		gasPriceRefund = takeFunds(availableRefund, gasPriceRefund)
		if err := transfer(&tx.From, &tx.FeeRefundAddr, gasPriceRefund); err != nil {
			glog.Error("failed to transfer gasPriceRefund", "err", err)
		}
		availableRefund.Add(availableRefund, withheldGasFunds)
		availableRefund.Add(availableRefund, withheldSubmissionFee)

		// emit RedeemScheduled event
		retryTxInner, err := retryable.MakeTx(
			underlyingTx.ChainId(),
			0,
			basefee,
			usergas,
			ticketId,
			tx.FeeRefundAddr,
			availableRefund,
			submissionFee,
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
			availableRefund,
			submissionFee,
		)
		if err != nil {
			glog.Error("failed to emit RedeemScheduled event", "err", err)
		}

		if evm.Config.Debug {
			redeem, err := util.PackArbRetryableTxRedeem(ticketId)
			if err == nil {
				tracingInfo.MockCall(redeem, usergas, from, types.ArbRetryableTxAddress, common.Big0)
			} else {
				glog.Error("failed to abi-encode auto-redeem", "err", err)
			}
		}

		return true, usergas, nil, ticketId.Bytes()
	case *types.ArbitrumRetryTx:

		// Transfer callvalue from escrow
		escrow := retryables.RetryableEscrowAddress(tx.TicketId)
		scenario := util.TracingBeforeEVM
		if err := util.TransferBalance(&escrow, &tx.From, tx.Value, evm, scenario, "escrow"); err != nil {
			return true, 0, err, nil
		}

		// The redeemer has pre-paid for this tx's gas
		prepaid := arbmath.BigMulByUint(evm.Context.BaseFee, tx.Gas)
		util.MintBalance(&tx.From, prepaid, evm, scenario, "prepaid")
		ticketId := tx.TicketId
		refundTo := tx.RefundTo
		p.CurrentRetryable = &ticketId
		p.CurrentRefundTo = &refundTo
	}
	return false, 0, nil, nil
}

func (p *TxProcessor) GasChargingHook(gasRemaining *uint64) (common.Address, error) {
	// Because a user pays a 1-dimensional gas price, we must re-express poster L1 calldata costs
	// as if the user was buying an equivalent amount of L2 compute gas. This hook determines what
	// that cost looks like, ensuring the user can pay and saving the result for later reference.

	var gasNeededToStartEVM uint64
	tipReceipient, _ := p.state.NetworkFeeAccount()
	basefee := p.evm.Context.BaseFee

	var poster common.Address
	if p.msg.RunMode() != types.MessageCommitMode {
		poster = l1pricing.BatchPosterAddress
	} else {
		poster = p.evm.Context.Coinbase
	}
	posterCost, calldataUnits := p.state.L1PricingState().PosterDataCost(p.msg, poster)
	if calldataUnits > 0 {
		p.state.Restrict(p.state.L1PricingState().AddToUnitsSinceUpdate(calldataUnits))
	}

	if p.msg.RunMode() == types.MessageGasEstimationMode {
		// Suggest the amount of gas needed for a given amount of ETH is higher in case of congestion.
		// This will help the user pad the total they'll pay in case the price rises a bit.
		// Note, reducing the poster cost will increase share the network fee gets, not reduce the total.

		minGasPrice, _ := p.state.L2PricingState().MinBaseFeeWei()

		adjustedPrice := arbmath.BigMulByFrac(basefee, 7, 8) // assume congestion
		if arbmath.BigLessThan(adjustedPrice, minGasPrice) {
			adjustedPrice = minGasPrice
		}
		basefee = adjustedPrice

		// Pad the L1 cost in case the L1 gas price rises
		posterCost = arbmath.BigMulByBips(posterCost, GasEstimationL1PricePadding)
	}

	if basefee.Sign() > 0 {
		// Since tips go to the network, and not to the poster, we use the basefee.
		// Note, this only determines the amount of gas bought, not the price per gas.

		p.posterGas = arbmath.BigToUintSaturating(arbmath.BigDiv(posterCost, basefee))
		p.PosterFee = arbmath.BigMulByUint(basefee, p.posterGas) // round down
		gasNeededToStartEVM = p.posterGas
	}

	if *gasRemaining < gasNeededToStartEVM {
		// the user couldn't pay for call data, so give up
		return tipReceipient, core.ErrIntrinsicGas
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
	return tipReceipient, nil
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
	networkFeeAccount, _ := p.state.NetworkFeeAccount()
	basefee := p.evm.Context.BaseFee
	scenario := util.TracingAfterEVM

	if gasLeft > p.msg.Gas() {
		panic("Tx somehow refunds gas after computation")
	}
	gasUsed := p.msg.Gas() - gasLeft

	if underlyingTx != nil && underlyingTx.Type() == types.ArbitrumRetryTxType {
		inner, _ := underlyingTx.GetInner().(*types.ArbitrumRetryTx)

		// undo Geth's refund to the From address
		gasRefund := arbmath.BigMulByUint(basefee, gasLeft)
		err := util.BurnBalance(&inner.From, gasRefund, p.evm, scenario, "undoRefund")
		if err != nil {
			log.Error("Uh oh, Geth didn't refund the user", inner.From, gasRefund)
		}

		maxRefund := new(big.Int).Set(inner.MaxRefund)
		refundNetworkFee := func(amount *big.Int) {
			const errLog = "network fee address doesn't have enough funds to give user refund"

			// Refund funds to the fee refund address without overdrafting the L1 deposit.
			toRefundAddr := takeFunds(maxRefund, amount)
			err = util.TransferBalance(&networkFeeAccount, &inner.RefundTo, toRefundAddr, p.evm, scenario, "refund")
			if err != nil {
				// Normally the network fee address should be holding any collected fees.
				// However, in theory, they could've been transfered out during the redeem attempt.
				// If the network fee address doesn't have the necessary balance, log an error and don't give a refund.
				log.Error(errLog, "err", err)
			}
			// Any extra refund can't be given to the fee refund address if it didn't come from the L1 deposit.
			// Instead, give the refund to the retryable from address.
			err = util.TransferBalance(&networkFeeAccount, &inner.From, arbmath.BigSub(amount, toRefundAddr), p.evm, scenario, "refund")
			if err != nil {
				log.Error(errLog, "err", err)
			}
		}

		if success {
			// If successful, refund the submission fee.
			refundNetworkFee(inner.SubmissionFeeRefund)
		} else {
			// The submission fee is still taken from the L1 deposit earlier, even if it's not refunded.
			takeFunds(maxRefund, inner.SubmissionFeeRefund)
		}
		// Conceptually, the gas charge is taken from the L1 deposit pool if possible.
		takeFunds(maxRefund, arbmath.BigMulByUint(basefee, gasUsed))
		// Refund any unused gas, without overdrafting the L1 deposit.
		refundNetworkFee(gasRefund)

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

	totalCost := arbmath.BigMul(basefee, arbmath.UintToBig(gasUsed)) // total cost = price of gas * gas burnt
	computeCost := arbmath.BigSub(totalCost, p.PosterFee)            // total cost = network's compute + poster's L1 costs
	if computeCost.Sign() < 0 {
		// Uh oh, there's a bug in our charging code.
		// Give all funds to the network account and continue.

		log.Error("total cost < poster cost", "gasUsed", gasUsed, "basefee", basefee, "posterFee", p.PosterFee)
		p.PosterFee = big.NewInt(0)
		computeCost = totalCost
	}

	purpose := "feeCollection"
	if p.state.ArbOSVersion() > 4 {
		infraFeeAccount, err := p.state.InfraFeeAccount()
		p.state.Restrict(err)
		if infraFeeAccount != (common.Address{}) {
			infraFee, err := p.state.L2PricingState().MinBaseFeeWei()
			p.state.Restrict(err)
			if arbmath.BigLessThan(basefee, infraFee) {
				infraFee = basefee
			}
			computeGas := arbmath.SaturatingUSub(gasUsed, p.posterGas)
			infraComputeCost := arbmath.BigMulByUint(infraFee, computeGas)
			util.MintBalance(&infraFeeAccount, infraComputeCost, p.evm, scenario, purpose)
			computeCost = arbmath.BigSub(computeCost, infraComputeCost)
		}
	}
	if arbmath.BigGreaterThan(computeCost, common.Big0) {
		util.MintBalance(&networkFeeAccount, computeCost, p.evm, scenario, purpose)
	}
	posterFeeDestination := l1pricing.L1PricerFundsPoolAddress
	if p.state.ArbOSVersion() < 2 {
		posterFeeDestination = p.evm.Context.Coinbase
	}
	util.MintBalance(&posterFeeDestination, p.PosterFee, p.evm, scenario, purpose)
	if p.state.ArbOSVersion() >= 10 {
		if _, err := p.state.L1PricingState().AddToL1FeesAvailable(p.PosterFee); err != nil {
			log.Error("failed to update L1FeesAvailable: ", "err", err)
		}
	}

	if p.msg.GasPrice().Sign() > 0 { // in tests, gas price could be 0
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
			event.MaxRefund,
			event.SubmissionFeeRefund,
		)
		scheduled = append(scheduled, types.NewTx(redeem))
	}
	return scheduled
}

func (p *TxProcessor) L1BlockNumber(blockCtx vm.BlockContext) (uint64, error) {
	if p.cachedL1BlockNumber != nil {
		return *p.cachedL1BlockNumber, nil
	}
	tracingInfo := util.NewTracingInfo(p.evm, p.msg.From(), arbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenSystemArbosState(p.evm.StateDB, tracingInfo, false)
	if err != nil {
		return 0, err
	}
	blockNum, err := state.Blockhashes().L1BlockNumber()
	if err != nil {
		return 0, err
	}
	p.cachedL1BlockNumber = &blockNum
	return blockNum, nil
}

func (p *TxProcessor) L1BlockHash(blockCtx vm.BlockContext, l1BlockNumber uint64) (common.Hash, error) {
	hash, cached := p.cachedL1BlockHashes[l1BlockNumber]
	if cached {
		return hash, nil
	}
	tracingInfo := util.NewTracingInfo(p.evm, p.msg.From(), arbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenSystemArbosState(p.evm.StateDB, tracingInfo, false)
	if err != nil {
		return common.Hash{}, err
	}
	hash, err = state.Blockhashes().BlockHash(l1BlockNumber)
	if err != nil {
		return common.Hash{}, err
	}
	p.cachedL1BlockHashes[l1BlockNumber] = hash
	return hash, nil
}

func (p *TxProcessor) DropTip() bool {
	version := p.state.ArbOSVersion()
	transaction := p.msg.UnderlyingTransaction()
	var enableTipFlag bool
	if version >= 11 && transaction.Type() == types.ArbitrumExtendedTxType {
		enableTipFlag = transaction.GetInner().(*types.ArbitrumExtendedTxData).EnableTipFlag()
	}
	return (version != 9 || p.delayedInbox) && !enableTipFlag
}

func (p *TxProcessor) GetPaidGasPrice() *big.Int {
	gasPrice := p.evm.GasPrice
	version := p.state.ArbOSVersion()
	if version != 9 {
		gasPrice = p.evm.Context.BaseFee
		if p.msg.RunMode() != types.MessageCommitMode && p.msg.GasFeeCap().Sign() == 0 {
			gasPrice.SetInt64(0) // gasprice zero behavior
		}
	}
	return gasPrice
}

func (p *TxProcessor) GasPriceOp(evm *vm.EVM) *big.Int {
	if p.state.ArbOSVersion() >= 3 {
		return p.GetPaidGasPrice()
	}
	return evm.GasPrice
}

func (p *TxProcessor) FillReceiptInfo(receipt *types.Receipt) {
	receipt.GasUsedForL1 = p.posterGas
}
