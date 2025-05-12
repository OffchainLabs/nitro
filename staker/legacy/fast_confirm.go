// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package legacystaker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/solgen/go/contractsgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/txbuilder"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type FastConfirmSafe struct {
	safe                      *contractsgen.Safe
	owners                    []common.Address
	threshold                 uint64
	fastConfirmNextNodeMethod abi.Method
	builder                   *txbuilder.Builder
	wallet                    ValidatorWalletInterface
	l1Reader                  *headerreader.HeaderReader
}

func NewFastConfirmSafe(
	callOpts *bind.CallOpts,
	fastConfirmSafeAddress common.Address,
	builder *txbuilder.Builder,
	wallet ValidatorWalletInterface,
	l1Reader *headerreader.HeaderReader,
) (*FastConfirmSafe, error) {
	fastConfirmSafe := &FastConfirmSafe{
		builder:  builder,
		wallet:   wallet,
		l1Reader: l1Reader,
	}
	safe, err := contractsgen.NewSafe(fastConfirmSafeAddress, wallet.L1Client())
	if err != nil {
		return nil, err
	}
	fastConfirmSafe.safe = safe
	owners, err := safe.GetOwners(callOpts)
	if err != nil {
		return nil, fmt.Errorf("calling getOwners: %w", err)
	}

	// This is needed because safe contract needs owners to be sorted.
	sort.Slice(owners, func(i, j int) bool {
		return owners[i].Cmp(owners[j]) < 0
	})
	fastConfirmSafe.owners = owners
	threshold, err := safe.GetThreshold(callOpts)
	if err != nil {
		return nil, fmt.Errorf("calling getThreshold: %w", err)
	}
	fastConfirmSafe.threshold = threshold.Uint64()
	rollupUserLogicAbi, err := rollupgen.RollupUserLogicMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	fastConfirmNextNodeMethod, ok := rollupUserLogicAbi.Methods["fastConfirmNextNode"]
	if !ok {
		return nil, errors.New("RollupUserLogic ABI missing fastConfirmNextNode method")
	}
	fastConfirmSafe.fastConfirmNextNodeMethod = fastConfirmNextNodeMethod
	return fastConfirmSafe, nil
}

func (f *FastConfirmSafe) tryFastConfirmation(ctx context.Context, blockHash common.Hash, sendRoot common.Hash, nodeHash common.Hash) error {
	if f.wallet.Address() == nil {
		return errors.New("fast confirmation requires a wallet which is not setup")
	}
	fastConfirmCallData, err := f.createFastConfirmCalldata(blockHash, sendRoot, nodeHash)
	if err != nil {
		return err
	}
	callOpts := &bind.CallOpts{Context: ctx}
	// Current nonce of the safe.
	nonce, err := f.safe.Nonce(callOpts)
	if err != nil {
		return err
	}
	// Hash of the safe transaction.
	safeTxHash, err := f.safe.GetTransactionHash(
		callOpts,
		f.wallet.RollupAddress(),
		big.NewInt(0),
		fastConfirmCallData,
		0,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		common.Address{},
		common.Address{},
		nonce,
	)
	if err != nil {
		return err
	}
	if !f.wallet.CanBatchTxs() {
		err = f.flushTransactions(ctx)
		if err != nil {
			return err
		}
	}

	alreadyApproved, err := f.safe.ApprovedHashes(&bind.CallOpts{Context: ctx}, *f.wallet.Address(), safeTxHash)
	if err != nil {
		return err
	}
	if alreadyApproved.Cmp(common.Big1) == 0 {
		log.Info("Already approved Safe tx hash for fast confirmation, checking if we can execute the Safe tx", "safeHash", safeTxHash, "nodeHash", nodeHash)
		_, err = f.checkApprovedHashAndExecTransaction(ctx, fastConfirmCallData, safeTxHash)
		return err
	}

	log.Info("Approving Safe tx hash to fast confirm", "safeHash", safeTxHash, "nodeHash", nodeHash)
	_, err = f.safe.ApproveHash(f.builder.Auth(ctx), safeTxHash)
	if err != nil {
		return err
	}
	if !f.wallet.CanBatchTxs() {
		err = f.flushTransactions(ctx)
		if err != nil {
			return err
		}
	}
	executedTx, err := f.checkApprovedHashAndExecTransaction(ctx, fastConfirmCallData, safeTxHash)
	if err != nil {
		return err
	}
	if executedTx {
		return nil
	}
	// If the transaction was not executed, we need to flush the transactions (for approve hash) and try again.
	// This is because the hash might have been approved by another wallet in the same block,
	// which might have led to a race condition.
	err = f.flushTransactions(ctx)
	if err != nil {
		return err
	}
	_, err = f.checkApprovedHashAndExecTransaction(ctx, fastConfirmCallData, safeTxHash)
	return err
}

func (f *FastConfirmSafe) flushTransactions(ctx context.Context) error {
	arbTx, err := f.builder.ExecuteTransactions(ctx)
	if err != nil {
		return err
	}
	if arbTx != nil {
		_, err = f.l1Reader.WaitForTxApproval(ctx, arbTx)
		if err == nil {
			log.Info("successfully executed staker transaction", "hash", arbTx.Hash())
		} else {
			return fmt.Errorf("error waiting for tx receipt: %w", err)
		}
	}
	return nil
}

func (f *FastConfirmSafe) createFastConfirmCalldata(
	blockHash common.Hash, sendRoot common.Hash, nodeHash common.Hash,
) ([]byte, error) {
	calldata, err := f.fastConfirmNextNodeMethod.Inputs.Pack(
		blockHash,
		sendRoot,
		nodeHash,
	)
	if err != nil {
		return nil, err
	}
	fullCalldata := append([]byte{}, f.fastConfirmNextNodeMethod.ID...)
	fullCalldata = append(fullCalldata, calldata...)
	return fullCalldata, nil
}

func (f *FastConfirmSafe) checkApprovedHashAndExecTransaction(ctx context.Context, fastConfirmCallData []byte, safeTxHash [32]byte) (bool, error) {
	if f.wallet.Address() == nil {
		return false, errors.New("wallet address is nil")
	}
	var signatures []byte
	approvedHashCount := uint64(0)
	for _, owner := range f.owners {
		var approved *big.Int
		// No need check if wallet has approved the hash,
		// since checkApprovedHashAndExecTransaction is called only after wallet has approved the hash.
		if *f.wallet.Address() == owner {
			approved = common.Big1
		} else {
			var err error
			approved, err = f.safe.ApprovedHashes(&bind.CallOpts{Context: ctx}, owner, safeTxHash)
			if err != nil {
				return false, err
			}
		}

		// If the owner has approved the hash, we add the signature to the transaction.
		// We add the signature in the format r, s, v.
		// We set v to 1, as it is the only possible value for a approved hash.
		// We set r to the owner's address.
		// We set s to the empty hash.
		// Refer to the Safe contract for more information.
		if approved.Cmp(common.Big1) == 0 {
			approvedHashCount++
			v := uint8(1)
			r := common.BytesToHash(owner.Bytes())
			s := common.Hash{}
			signatures = append(signatures, r.Bytes()...)
			signatures = append(signatures, s.Bytes()...)
			signatures = append(signatures, v)
		}
	}
	if approvedHashCount >= f.threshold {
		log.Info("Executing Safe tx to fast confirm", "safeHash", safeTxHash)
		_, err := f.safe.ExecTransaction(
			f.builder.Auth(ctx),
			f.wallet.RollupAddress(),
			big.NewInt(0),
			fastConfirmCallData,
			0,
			big.NewInt(0),
			big.NewInt(0),
			big.NewInt(0),
			common.Address{},
			common.Address{},
			signatures,
		)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	log.Info("Not enough Safe tx approvals yet to fast confirm", "safeHash", safeTxHash)
	return false, nil
}
