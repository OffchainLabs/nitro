// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/util/arbmath"
	am "github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

type L1PricingState struct {
	storage *storage.Storage

	// parameters
	batchPosterTable   *BatchPostersTable
	payRewardsTo       storage.StorageBackedAddress
	equilibrationUnits storage.StorageBackedBigUint
	inertia            storage.StorageBackedUint64
	perUnitReward      storage.StorageBackedUint64
	// variables
	lastUpdateTime         storage.StorageBackedUint64 // timestamp of the last update from L1 that we processed
	fundsDueForRewards     storage.StorageBackedBigInt
	brotliCompressionLevel storage.StorageBackedUint64 // brotli compression level used for pricing
	// funds collected since update are recorded as the balance in account L1PricerFundsPoolAddress
	unitsSinceUpdate     storage.StorageBackedUint64  // calldata units collected for since last update
	pricePerUnit         storage.StorageBackedBigUint // current price per calldata unit
	lastSurplus          storage.StorageBackedBigInt  // introduced in ArbOS version 2
	perBatchGasCost      storage.StorageBackedInt64   // introduced in ArbOS version 3
	amortizedCostCapBips storage.StorageBackedUint64  // in basis points; introduced in ArbOS version 3
	l1FeesAvailable      storage.StorageBackedBigUint
}

var (
	BatchPosterTableKey      = []byte{0}
	BatchPosterAddress       = common.HexToAddress("0xA4B000000000000000000073657175656e636572")
	BatchPosterPayToAddress  = BatchPosterAddress
	L1PricerFundsPoolAddress = common.HexToAddress("0xA4B00000000000000000000000000000000000f6")

	ErrInvalidTime = errors.New("invalid timestamp")
)

const (
	payRewardsToOffset uint64 = iota
	equilibrationUnitsOffset
	inertiaOffset
	perUnitRewardOffset
	lastUpdateTimeOffset
	fundsDueForRewardsOffset
	brotliCompressionLevelOffset
	unitsSinceOffset
	pricePerUnitOffset
	lastSurplusOffset
	perBatchGasCostOffset
	amortizedCostCapBipsOffset
	l1FeesAvailableOffset
)

const (
	InitialInertia                = 10
	InitialPerUnitReward          = 10
	InitialPerBatchGasCostV6      = 100_000
	InitialPerBatchGasCostV12     = 210_000 // overriden as part of the upgrade
	InitialBrotliCompressionLevel = 0
)

// one minute at 100000 bytes / sec
var InitialEquilibrationUnitsV0 = arbmath.UintToBig(60 * params.TxDataNonZeroGasEIP2028 * 100000)
var InitialEquilibrationUnitsV6 = arbmath.UintToBig(params.TxDataNonZeroGasEIP2028 * 10000000)

func InitializeL1PricingState(sto *storage.Storage, initialRewardsRecipient common.Address, initialL1BaseFee *big.Int) error {
	bptStorage := sto.OpenSubStorage(BatchPosterTableKey)
	if err := InitializeBatchPostersTable(bptStorage); err != nil {
		return err
	}
	bpTable := OpenBatchPostersTable(bptStorage)
	if _, err := bpTable.AddPoster(BatchPosterAddress, BatchPosterPayToAddress); err != nil {
		return err
	}
	if err := sto.SetByUint64(payRewardsToOffset, util.AddressToHash(initialRewardsRecipient)); err != nil {
		return err
	}
	equilibrationUnits := sto.OpenStorageBackedBigUint(equilibrationUnitsOffset)
	if err := equilibrationUnits.SetChecked(InitialEquilibrationUnitsV0); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(inertiaOffset, InitialInertia); err != nil {
		return err
	}
	fundsDueForRewards := sto.OpenStorageBackedBigInt(fundsDueForRewardsOffset)
	if err := fundsDueForRewards.SetChecked(common.Big0); err != nil {
		return err
	}
	if err := sto.SetUint64ByUint64(perUnitRewardOffset, InitialPerUnitReward); err != nil {
		return err
	}
	pricePerUnit := sto.OpenStorageBackedBigInt(pricePerUnitOffset)
	if err := pricePerUnit.SetSaturatingWithWarning(initialL1BaseFee, "initial L1 base fee (storing in price per unit)"); err != nil {
		return err
	}
	brotliCompressionLevel := sto.OpenStorageBackedUint64(brotliCompressionLevelOffset)
	if err := brotliCompressionLevel.Set(InitialBrotliCompressionLevel); err != nil {
		return err
	}
	return nil
}

func OpenL1PricingState(sto *storage.Storage) *L1PricingState {
	return &L1PricingState{
		sto,
		OpenBatchPostersTable(sto.OpenSubStorage(BatchPosterTableKey)),
		sto.OpenStorageBackedAddress(payRewardsToOffset),
		sto.OpenStorageBackedBigUint(equilibrationUnitsOffset),
		sto.OpenStorageBackedUint64(inertiaOffset),
		sto.OpenStorageBackedUint64(perUnitRewardOffset),
		sto.OpenStorageBackedUint64(lastUpdateTimeOffset),
		sto.OpenStorageBackedBigInt(fundsDueForRewardsOffset),
		sto.OpenStorageBackedUint64(brotliCompressionLevelOffset),
		sto.OpenStorageBackedUint64(unitsSinceOffset),
		sto.OpenStorageBackedBigUint(pricePerUnitOffset),
		sto.OpenStorageBackedBigInt(lastSurplusOffset),
		sto.OpenStorageBackedInt64(perBatchGasCostOffset),
		sto.OpenStorageBackedUint64(amortizedCostCapBipsOffset),
		sto.OpenStorageBackedBigUint(l1FeesAvailableOffset),
	}
}

func (ps *L1PricingState) BatchPosterTable() *BatchPostersTable {
	return ps.batchPosterTable
}

func (ps *L1PricingState) PayRewardsTo() (common.Address, error) {
	return ps.payRewardsTo.Get()
}

func (ps *L1PricingState) SetPayRewardsTo(addr common.Address) error {
	return ps.payRewardsTo.Set(addr)
}

func (ps *L1PricingState) GetRewardsRecepient() (common.Address, error) {
	return ps.payRewardsTo.Get()
}

func (ps *L1PricingState) EquilibrationUnits() (*big.Int, error) {
	return ps.equilibrationUnits.Get()
}

func (ps *L1PricingState) SetEquilibrationUnits(equilUnits *big.Int) error {
	return ps.equilibrationUnits.SetChecked(equilUnits)
}

func (ps *L1PricingState) Inertia() (uint64, error) {
	return ps.inertia.Get()
}

func (ps *L1PricingState) SetInertia(inertia uint64) error {
	return ps.inertia.Set(inertia)
}

func (ps *L1PricingState) PerUnitReward() (uint64, error) {
	return ps.perUnitReward.Get()
}

func (ps *L1PricingState) SetPerUnitReward(weiPerUnit uint64) error {
	return ps.perUnitReward.Set(weiPerUnit)
}

func (ps *L1PricingState) GetRewardsRate() (uint64, error) {
	return ps.perUnitReward.Get()
}

func (ps *L1PricingState) LastUpdateTime() (uint64, error) {
	return ps.lastUpdateTime.Get()
}

func (ps *L1PricingState) SetLastUpdateTime(t uint64) error {
	return ps.lastUpdateTime.Set(t)
}

func (ps *L1PricingState) FundsDueForRewards() (*big.Int, error) {
	return ps.fundsDueForRewards.Get()
}

func (ps *L1PricingState) SetFundsDueForRewards(amt *big.Int) error {
	return ps.fundsDueForRewards.SetSaturatingWithWarning(amt, "L1 pricer funds due for rewards")

}

func (ps *L1PricingState) UnitsSinceUpdate() (uint64, error) {
	return ps.unitsSinceUpdate.Get()
}

func (ps *L1PricingState) SetUnitsSinceUpdate(units uint64) error {
	return ps.unitsSinceUpdate.Set(units)
}

func (ps *L1PricingState) LastSurplus() (*big.Int, error) {
	return ps.lastSurplus.Get()
}

func (ps *L1PricingState) SetLastSurplus(val *big.Int, arbosVersion uint64) error {
	if arbosVersion < 7 {
		return ps.lastSurplus.Set_preVersion7(val)
	}
	return ps.lastSurplus.SetSaturatingWithWarning(val, "L1 pricer last surplus")
}

func (ps *L1PricingState) AddToUnitsSinceUpdate(units uint64) error {
	oldUnits, err := ps.unitsSinceUpdate.Get()
	if err != nil {
		return err
	}
	return ps.unitsSinceUpdate.Set(oldUnits + units)
}

func (ps *L1PricingState) PricePerUnit() (*big.Int, error) {
	return ps.pricePerUnit.Get()
}

func (ps *L1PricingState) SetPricePerUnit(price *big.Int) error {
	return ps.pricePerUnit.SetChecked(price)
}

func (ps *L1PricingState) PerBatchGasCost() (int64, error) {
	return ps.perBatchGasCost.Get()
}

func (ps *L1PricingState) SetPerBatchGasCost(cost int64) error {
	return ps.perBatchGasCost.Set(cost)
}

func (ps *L1PricingState) AmortizedCostCapBips() (uint64, error) {
	return ps.amortizedCostCapBips.Get()
}

func (ps *L1PricingState) SetAmortizedCostCapBips(cap uint64) error {
	return ps.amortizedCostCapBips.Set(cap)
}

func (ps *L1PricingState) L1FeesAvailable() (*big.Int, error) {
	return ps.l1FeesAvailable.Get()
}

func (ps *L1PricingState) SetL1FeesAvailable(val *big.Int) error {
	return ps.l1FeesAvailable.SetChecked(val)
}

func (ps *L1PricingState) BrotliCompressionLevel() (uint64, error) {
	return ps.brotliCompressionLevel.Get()
}

func (ps *L1PricingState) SetBrotliCompressionLevel(val uint64) error {
	if val <= arbcompress.LEVEL_WELL {
		return ps.brotliCompressionLevel.Set(val)
	}
	return errors.New("invalid brotli compression level")
}

func (ps *L1PricingState) AddToL1FeesAvailable(delta *big.Int) (*big.Int, error) {
	old, err := ps.L1FeesAvailable()
	if err != nil {
		return nil, err
	}
	new := new(big.Int).Add(old, delta)
	if err := ps.SetL1FeesAvailable(new); err != nil {
		return nil, err
	}
	return new, nil
}

func (ps *L1PricingState) TransferFromL1FeesAvailable(
	recipient common.Address,
	amount *big.Int,
	evm *vm.EVM,
	scenario util.TracingScenario,
	purpose string,
) (*big.Int, error) {
	if err := util.TransferBalance(&L1PricerFundsPoolAddress, &recipient, amount, evm, scenario, purpose); err != nil {
		return nil, err
	}
	old, err := ps.L1FeesAvailable()
	if err != nil {
		return nil, err
	}
	updated := new(big.Int).Sub(old, amount)
	if updated.Sign() < 0 {
		return nil, core.ErrInsufficientFunds
	}
	if err := ps.SetL1FeesAvailable(updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// UpdateForBatchPosterSpending updates the pricing model based on a payment by a batch poster
func (ps *L1PricingState) UpdateForBatchPosterSpending(
	statedb vm.StateDB,
	evm *vm.EVM,
	arbosVersion uint64,
	updateTime, currentTime uint64,
	batchPoster common.Address,
	weiSpent *big.Int,
	l1Basefee *big.Int,
	scenario util.TracingScenario,
) error {
	if arbosVersion < 10 {
		return ps._preversion10_UpdateForBatchPosterSpending(statedb, evm, arbosVersion, updateTime, currentTime, batchPoster, weiSpent, l1Basefee, scenario)
	}

	batchPosterTable := ps.BatchPosterTable()
	posterState, err := batchPosterTable.OpenPoster(batchPoster, true)
	if err != nil {
		return err
	}

	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return err
	}

	l1FeesAvailable, err := ps.L1FeesAvailable()
	if err != nil {
		return err
	}

	// compute allocation fraction -- will allocate updateTimeDelta/timeDelta fraction of units and funds to this update
	lastUpdateTime, err := ps.LastUpdateTime()
	if err != nil {
		return err
	}
	if lastUpdateTime == 0 && updateTime > 0 { // it's the first update, so there isn't a last update time
		lastUpdateTime = updateTime - 1
	}
	if updateTime > currentTime || updateTime < lastUpdateTime {
		return ErrInvalidTime
	}
	allocationNumerator := updateTime - lastUpdateTime
	allocationDenominator := currentTime - lastUpdateTime
	if allocationDenominator == 0 {
		allocationNumerator = 1
		allocationDenominator = 1
	}

	// allocate units to this update
	unitsSinceUpdate, err := ps.UnitsSinceUpdate()
	if err != nil {
		return err
	}
	unitsAllocated := am.SaturatingUMul(unitsSinceUpdate, allocationNumerator) / allocationDenominator
	unitsSinceUpdate -= unitsAllocated
	if err := ps.SetUnitsSinceUpdate(unitsSinceUpdate); err != nil {
		return err
	}

	// impose cap on amortized cost, if there is one
	if arbosVersion >= 3 {
		amortizedCostCapBips, err := ps.AmortizedCostCapBips()
		if err != nil {
			return err
		}
		if amortizedCostCapBips != 0 {
			weiSpentCap := am.BigMulByBips(
				am.BigMulByUint(l1Basefee, unitsAllocated),
				am.SaturatingCastToBips(amortizedCostCapBips),
			)
			if am.BigLessThan(weiSpentCap, weiSpent) {
				// apply the cap on assignment of amortized cost;
				// the difference will be a loss for the batch poster
				weiSpent = weiSpentCap
			}
		}
	}

	dueToPoster, err := posterState.FundsDue()
	if err != nil {
		return err
	}
	err = posterState.SetFundsDue(am.BigAdd(dueToPoster, weiSpent))
	if err != nil {
		return err
	}
	perUnitReward, err := ps.PerUnitReward()
	if err != nil {
		return err
	}
	fundsDueForRewards = am.BigAdd(fundsDueForRewards, am.BigMulByUint(am.UintToBig(unitsAllocated), perUnitReward))
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}

	// pay rewards, as much as possible
	paymentForRewards := am.BigMulByUint(am.UintToBig(perUnitReward), unitsAllocated)
	if am.BigLessThan(l1FeesAvailable, paymentForRewards) {
		paymentForRewards = l1FeesAvailable
	}
	fundsDueForRewards = am.BigSub(fundsDueForRewards, paymentForRewards)
	if err := ps.SetFundsDueForRewards(fundsDueForRewards); err != nil {
		return err
	}
	payRewardsTo, err := ps.PayRewardsTo()
	if err != nil {
		return err
	}
	l1FeesAvailable, err = ps.TransferFromL1FeesAvailable(
		payRewardsTo, paymentForRewards, evm, scenario, "batchPosterReward",
	)
	if err != nil {
		return err
	}

	// settle up payments owed to the batch poster, as much as possible
	balanceDueToPoster, err := posterState.FundsDue()
	if err != nil {
		return err
	}
	balanceToTransfer := balanceDueToPoster
	if am.BigLessThan(l1FeesAvailable, balanceToTransfer) {
		balanceToTransfer = l1FeesAvailable
	}
	if balanceToTransfer.Sign() > 0 {
		addrToPay, err := posterState.PayTo()
		if err != nil {
			return err
		}
		l1FeesAvailable, err = ps.TransferFromL1FeesAvailable(
			addrToPay, balanceToTransfer, evm, scenario, "batchPosterRefund",
		)
		if err != nil {
			return err
		}
		balanceDueToPoster = am.BigSub(balanceDueToPoster, balanceToTransfer)
		err = posterState.SetFundsDue(balanceDueToPoster)
		if err != nil {
			return err
		}
	}

	// update time
	if err := ps.SetLastUpdateTime(updateTime); err != nil {
		return err
	}

	// adjust the price
	if unitsAllocated > 0 {
		totalFundsDue, err := batchPosterTable.TotalFundsDue()
		if err != nil {
			return err
		}
		fundsDueForRewards, err = ps.FundsDueForRewards()
		if err != nil {
			return err
		}
		surplus := am.BigSub(l1FeesAvailable, am.BigAdd(totalFundsDue, fundsDueForRewards))

		inertia, err := ps.Inertia()
		if err != nil {
			return err
		}
		equilUnits, err := ps.EquilibrationUnits()
		if err != nil {
			return err
		}
		inertiaUnits := am.BigDivByUint(equilUnits, inertia)
		price, err := ps.PricePerUnit()
		if err != nil {
			return err
		}

		allocPlusInert := am.BigAddByUint(inertiaUnits, unitsAllocated)
		oldSurplus, err := ps.LastSurplus()
		if err != nil {
			return err
		}

		desiredDerivative := am.BigDiv(new(big.Int).Neg(surplus), equilUnits)
		actualDerivative := am.BigDivByUint(am.BigSub(surplus, oldSurplus), unitsAllocated)
		changeDerivativeBy := am.BigSub(desiredDerivative, actualDerivative)
		priceChange := am.BigDiv(am.BigMulByUint(changeDerivativeBy, unitsAllocated), allocPlusInert)

		if err := ps.SetLastSurplus(surplus, arbosVersion); err != nil {
			return err
		}
		newPrice := am.BigAdd(price, priceChange)
		if newPrice.Sign() < 0 {
			newPrice = common.Big0
		}
		if err := ps.SetPricePerUnit(newPrice); err != nil {
			return err
		}
	}
	return nil
}

func (ps *L1PricingState) getPosterUnitsWithoutCache(tx *types.Transaction, posterAddr common.Address) uint64 {

	if posterAddr != BatchPosterAddress {
		return 0
	}
	txBytes, merr := tx.MarshalBinary()
	txType := tx.Type()
	if !util.TxTypeHasPosterCosts(txType) || merr != nil {
		return 0
	}

	level, err := ps.BrotliCompressionLevel()
	if err != nil {
		panic(fmt.Sprintf("failed to get brotli compression level: %v", err))
	}
	l1Bytes, err := byteCountAfterBrotliLevel(txBytes, int(level))
	if err != nil {
		panic(fmt.Sprintf("failed to compress tx: %v", err))
	}
	return l1Bytes * params.TxDataNonZeroGasEIP2028
}

// GetPosterInfo returns the poster cost and the calldata units for a transaction
func (ps *L1PricingState) GetPosterInfo(tx *types.Transaction, poster common.Address) (*big.Int, uint64) {
	if poster != BatchPosterAddress {
		return common.Big0, 0
	}
	units := atomic.LoadUint64(&tx.CalldataUnits)
	if units == 0 {
		units = ps.getPosterUnitsWithoutCache(tx, poster)
		atomic.StoreUint64(&tx.CalldataUnits, units)
	}

	// Approximate the l1 fee charged for posting this tx's calldata
	pricePerUnit, _ := ps.PricePerUnit()
	return am.BigMulByUint(pricePerUnit, units), units
}

// We don't have the full tx in gas estimation, so we assume it might be a bit bigger in practice.
const estimationPaddingUnits = 16 * params.TxDataNonZeroGasEIP2028
const estimationPaddingBasisPoints = 100

var randomNonce = binary.BigEndian.Uint64(crypto.Keccak256([]byte("Nonce"))[:8])
var randomGasTipCap = new(big.Int).SetBytes(crypto.Keccak256([]byte("GasTipCap"))[:4])
var randomGasFeeCap = new(big.Int).SetBytes(crypto.Keccak256([]byte("GasFeeCap"))[:4])
var RandomGas = uint64(binary.BigEndian.Uint32(crypto.Keccak256([]byte("Gas"))[:4]))
var randV = arbmath.BigMulByUint(params.ArbitrumOneChainConfig().ChainID, 3)
var randR = crypto.Keccak256Hash([]byte("R")).Big()
var randS = crypto.Keccak256Hash([]byte("S")).Big()

// The returned tx will be invalid, likely for a number of reasons such as an invalid signature.
// It's only used to check how large it is after brotli level 0 compression.
func makeFakeTxForMessage(message *core.Message) *types.Transaction {
	nonce := message.Nonce
	if nonce == 0 {
		nonce = randomNonce
	}
	gasTipCap := message.GasTipCap
	if gasTipCap.Sign() == 0 {
		gasTipCap = randomGasTipCap
	}
	gasFeeCap := message.GasFeeCap
	if gasFeeCap.Sign() == 0 {
		gasFeeCap = randomGasFeeCap
	}
	// During gas estimation, we don't want the gas limit variability to change the L1 cost.
	gas := message.GasLimit
	if gas == 0 || message.TxRunMode == core.MessageGasEstimationMode {
		gas = RandomGas
	}
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:      nonce,
		GasTipCap:  gasTipCap,
		GasFeeCap:  gasFeeCap,
		Gas:        gas,
		To:         message.To,
		Value:      message.Value,
		Data:       message.Data,
		AccessList: message.AccessList,
		V:          randV,
		R:          randR,
		S:          randS,
	})
}

func (ps *L1PricingState) PosterDataCost(message *core.Message, poster common.Address) (*big.Int, uint64) {
	tx := message.Tx
	if tx != nil {
		return ps.GetPosterInfo(tx, poster)
	}

	// Otherwise, we don't have an underlying transaction, so we're likely in gas estimation.
	// We'll instead make a fake tx from the message info we do have, and then pad our cost a bit to be safe.
	tx = makeFakeTxForMessage(message)
	units := ps.getPosterUnitsWithoutCache(tx, poster)
	units = arbmath.UintMulByBips(units+estimationPaddingUnits, arbmath.OneInBips+estimationPaddingBasisPoints)
	pricePerUnit, _ := ps.PricePerUnit()
	return am.BigMulByUint(pricePerUnit, units), units
}

func byteCountAfterBrotliLevel(input []byte, level int) (uint64, error) {
	compressed, err := arbcompress.CompressFast(input, level)
	if err != nil {
		return 0, err
	}
	return uint64(len(compressed)), nil
}
