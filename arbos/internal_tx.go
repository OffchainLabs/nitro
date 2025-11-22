// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func InternalTxStartBlock(
	chainId,
	l1BaseFee *big.Int,
	l1BlockNum uint64,
	header,
	lastHeader *types.Header,
) *types.ArbitrumInternalTx {

	l2BlockNum := header.Number.Uint64()
	timePassed := header.Time - lastHeader.Time

	if l1BaseFee == nil {
		l1BaseFee = big.NewInt(0)
	}
	data, err := util.PackInternalTxDataStartBlock(l1BaseFee, l1BlockNum, l2BlockNum, timePassed)
	if err != nil {
		panic(fmt.Sprintf("Failed to pack internal tx %v", err))
	}
	return &types.ArbitrumInternalTx{
		ChainId: chainId,
		Data:    data,
	}
}

// In case floor_gas is used - this negates most of the difference between calldata and raw batch
// Raw batch has a 40-byte header that didn't come from calldata (5 uint64s)
// Calldata for the addSequencerL2BatchFromOrigin call in SequencerInbox, has a function selector
// and 5 additional fields that don't appear in the raw batch.
//
// Token count for the additional fields in calldata:
// 4*4 - 1 function selector (4 non-zero bytes)
// 4*24 - 4 fields fit in a uint64 - differ only by padding of 24 zero-bytes each
// 4*12 + 12 - 1 address field, so has about 12 additional nonzero bytes + 12 zero bytes for padding
// Total: 172
// This is not exact since most uint64s also have zeroes, and batch poster may use another function,
// but it doesn't need to be exact
const FloorGasAdditionalTokens uint64 = 172

func ApplyInternalTxUpdate(tx *types.ArbitrumInternalTx, state *arbosState.ArbosState, evm *vm.EVM) error {
	if len(tx.Data) < 4 {
		return fmt.Errorf("internal tx data is too short (only %v bytes, at least 4 required)", len(tx.Data))
	}
	selector := *(*[4]byte)(tx.Data[:4])
	switch selector {
	case InternalTxStartBlockMethodID:
		inputs, err := util.UnpackInternalTxDataStartBlock(tx.Data)
		if err != nil {
			return err
		}

		var prevHash common.Hash
		if evm.Context.BlockNumber.Sign() > 0 {
			prevHash = evm.Context.GetHash(evm.Context.BlockNumber.Uint64() - 1)
		}
		// For ArbOS versions >= 40 we need to call ProcessParentBlockHash to fill
		// the historyStorage with the block hash to support EIP-2935.
		if state.ArbOSVersion() >= params.ArbosVersion_40 {
			core.ProcessParentBlockHash(prevHash, evm)
		}
		l1BlockNumber := util.SafeMapGet[uint64](inputs, "l1BlockNumber")
		timePassed := util.SafeMapGet[uint64](inputs, "timePassed")
		if state.ArbOSVersion() < params.ArbosVersion_3 {
			// (incorrectly) use the L2 block number instead
			timePassed = util.SafeMapGet[uint64](inputs, "l2BlockNumber")
		}
		if state.ArbOSVersion() < params.ArbosVersion_8 {
			// in old versions we incorrectly used an L1 block number one too high
			l1BlockNumber++
		}

		oldL1BlockNumber, err := state.Blockhashes().L1BlockNumber()
		state.Restrict(err)

		if l1BlockNumber > oldL1BlockNumber {
			state.Restrict(state.Blockhashes().RecordNewL1Block(l1BlockNumber-1, prevHash, state.ArbOSVersion()))
		}

		currentTime := evm.Context.Time

		// Try to reap 2 retryables
		_ = state.RetryableState().TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)
		_ = state.RetryableState().TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)

		state.L2PricingState().UpdatePricingModel(timePassed)

		return state.UpgradeArbosVersionIfNecessary(currentTime, evm.StateDB, evm.ChainConfig())
	case InternalTxBatchPostingReportMethodID:
		inputs, err := util.UnpackInternalTxDataBatchPostingReport(tx.Data)
		if err != nil {
			return err
		}
		batchTimestamp := util.SafeMapGet[*big.Int](inputs, "batchTimestamp")
		batchPosterAddress := util.SafeMapGet[common.Address](inputs, "batchPosterAddress")
		batchDataGas := util.SafeMapGet[uint64](inputs, "batchDataGas")
		l1BaseFeeWei := util.SafeMapGet[*big.Int](inputs, "l1BaseFeeWei")

		l1p := state.L1PricingState()
		perBatchGas, err := l1p.PerBatchGasCost()
		if err != nil {
			log.Warn("L1Pricing PerBatchGas failed", "err", err)
		}
		gasSpent := arbmath.SaturatingAdd(perBatchGas, arbmath.SaturatingCast[int64](batchDataGas))
		weiSpent := arbmath.BigMulByUint(l1BaseFeeWei, arbmath.SaturatingUCast[uint64](gasSpent))
		err = l1p.UpdateForBatchPosterSpending(
			evm.StateDB,
			evm,
			state.ArbOSVersion(),
			batchTimestamp.Uint64(),
			evm.Context.Time,
			batchPosterAddress,
			weiSpent,
			l1BaseFeeWei,
			util.TracingDuringEVM,
		)
		if err != nil {
			log.Warn("L1Pricing UpdateForSequencerSpending failed", "err", err)
		}
		return nil
	case InternalTxBatchPostingReportV2MethodID:
		inputs, err := util.UnpackInternalTxDataBatchPostingReportV2(tx.Data)
		if err != nil {
			return err
		}
		batchTimestamp := util.SafeMapGet[*big.Int](inputs, "batchTimestamp")
		batchPosterAddress := util.SafeMapGet[common.Address](inputs, "batchPosterAddress")
		batchCalldataLength := util.SafeMapGet[uint64](inputs, "batchCalldataLength")
		batchCalldataNonZeros := util.SafeMapGet[uint64](inputs, "batchCalldataNonZeros")
		batchExtraGas := util.SafeMapGet[uint64](inputs, "batchExtraGas")
		l1BaseFeeWei := util.SafeMapGet[*big.Int](inputs, "l1BaseFeeWei")

		gasSpent := arbostypes.LegacyCostForStats(&arbostypes.BatchDataStats{
			Length:   batchCalldataLength,
			NonZeros: batchCalldataNonZeros,
		})

		gasSpent = arbmath.SaturatingUAdd(gasSpent, batchExtraGas)

		l1p := state.L1PricingState()

		perBatchGas, err := l1p.PerBatchGasCost()
		if err != nil {
			log.Warn("L1Pricing PerBatchGas failed", "err", err)
		}
		gasSpent = arbmath.SaturatingUAdd(gasSpent, arbmath.SaturatingUCast[uint64](perBatchGas))

		if state.ArbOSVersion() >= params.ArbosVersion_50 {
			gasFloorPerToken, err := l1p.ParentGasFloorPerToken()
			if err != nil {
				log.Warn("failed reading gasFloorPerToken", "err", err)
			}
			floorGasSpent := gasFloorPerToken*(batchCalldataLength+batchCalldataNonZeros*3+FloorGasAdditionalTokens) + params.TxGas
			if floorGasSpent > gasSpent {
				gasSpent = floorGasSpent
			}
		}

		weiSpent := arbmath.BigMulByUint(l1BaseFeeWei, gasSpent)
		err = l1p.UpdateForBatchPosterSpending(
			evm.StateDB,
			evm,
			state.ArbOSVersion(),
			batchTimestamp.Uint64(),
			evm.Context.Time,
			batchPosterAddress,
			weiSpent,
			l1BaseFeeWei,
			util.TracingDuringEVM,
		)
		if err != nil {
			log.Warn("L1Pricing UpdateForSequencerSpending failed (v2 report)", "err", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown internal tx method selector: %v", hex.EncodeToString(tx.Data[:4]))
	}
}
