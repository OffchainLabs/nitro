package solimpl

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/runtime"
	"github.com/offchainlabs/nitro/solgen/go/contractsgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

type FastConfirmSafe struct {
	safe                       *contractsgen.Safe
	owners                     []common.Address
	threshold                  uint64
	fastConfirmAssertionMethod abi.Method
	assertionChain             *AssertionChain
}

func NewFastConfirmSafe(
	callOpts *bind.CallOpts,
	fastConfirmSafeAddress common.Address,
	assertionChain *AssertionChain,
) (*FastConfirmSafe, error) {
	fastConfirmSafe := &FastConfirmSafe{
		assertionChain: assertionChain,
	}
	safe, err := retry.UntilSucceeds(callOpts.Context, func() (*contractsgen.Safe, error) {
		return contractsgen.NewSafe(fastConfirmSafeAddress, assertionChain.backend)
	})
	if err != nil {
		return nil, err
	}
	fastConfirmSafe.safe = safe
	owners, err := retry.UntilSucceeds(callOpts.Context, func() ([]common.Address, error) {
		return safe.GetOwners(callOpts)
	})
	if err != nil {
		return nil, fmt.Errorf("calling getOwners: %w", err)
	}

	// This is needed because safe contract needs owners to be sorted.
	sort.Slice(owners, func(i, j int) bool {
		return owners[i].Cmp(owners[j]) < 0
	})
	fastConfirmSafe.owners = owners
	threshold, err := retry.UntilSucceeds(callOpts.Context, func() (*big.Int, error) {
		return safe.GetThreshold(callOpts)
	})
	if err != nil {
		return nil, fmt.Errorf("calling getThreshold: %w", err)
	}
	fastConfirmSafe.threshold = threshold.Uint64()
	rollupUserLogicAbi, err := rollupgen.RollupUserLogicMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	fastConfirmAssertionMethod, ok := rollupUserLogicAbi.Methods["fastConfirmAssertion"]
	if !ok {
		return nil, errors.New("RollupUserLogic ABI missing fastConfirmNextNode method")
	}
	fastConfirmSafe.fastConfirmAssertionMethod = fastConfirmAssertionMethod
	return fastConfirmSafe, nil
}

func (f *FastConfirmSafe) fastConfirmAssertion(ctx context.Context, assertionCreationInfo *protocol.AssertionCreatedInfo) (bool, error) {
	fastConfirmCallData, err := f.createFastConfirmCalldata(assertionCreationInfo)
	if err != nil {
		return false, err
	}
	// Current nonce of the safe.
	callOpts := f.assertionChain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx})
	nonce, err := retry.UntilSucceeds(callOpts.Context, func() (*big.Int, error) {
		return f.safe.Nonce(callOpts)
	})
	if err != nil {
		return false, err
	}
	// Hash of the safe transaction.
	safeTxHash, err := retry.UntilSucceeds(callOpts.Context, func() (common.Hash, error) {
		return f.safe.GetTransactionHash(
			callOpts,
			f.assertionChain.rollupAddr,
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
	})
	if err != nil {
		return false, err
	}
	alreadyApproved, err := retry.UntilSucceeds(callOpts.Context, func() (*big.Int, error) {
		return f.safe.ApprovedHashes(callOpts, f.assertionChain.StakerAddress(), safeTxHash)
	})
	if err != nil {
		return false, err
	}
	if alreadyApproved.Cmp(common.Big1) == 0 {
		log.Info("Already approved Safe tx hash for fast confirmation, checking if we can execute the Safe tx", "safeHash", safeTxHash, "assertionHash", assertionCreationInfo.AssertionHash)
		return f.checkApprovedHashAndExecTransaction(ctx, callOpts, fastConfirmCallData, safeTxHash)
	}

	log.Info("Approving Safe tx hash to fast confirm", "safeHash", safeTxHash, "assertionHash", assertionCreationInfo.AssertionHash)
	receipt, err := f.assertionChain.transact(ctx, f.assertionChain.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return f.safe.ApproveHash(opts, safeTxHash)
	})
	if err != nil {
		return false, err
	}
	if len(receipt.Logs) == 0 {
		return false, errors.New("no logs observed from hash approval")
	}
	return f.checkApprovedHashAndExecTransaction(ctx, callOpts, fastConfirmCallData, safeTxHash)

}

func (f *FastConfirmSafe) createFastConfirmCalldata(
	assertionCreationInfo *protocol.AssertionCreatedInfo,
) ([]byte, error) {
	calldata, err := f.fastConfirmAssertionMethod.Inputs.Pack(
		assertionCreationInfo.AssertionHash.Hash,
		assertionCreationInfo.ParentAssertionHash.Hash,
		assertionCreationInfo.AfterState,
		assertionCreationInfo.AfterInboxBatchAcc,
	)
	if err != nil {
		return nil, err
	}
	fullCalldata := append([]byte{}, f.fastConfirmAssertionMethod.ID...)
	fullCalldata = append(fullCalldata, calldata...)
	return fullCalldata, nil
}

func (f *FastConfirmSafe) checkApprovedHashAndExecTransaction(
	ctx context.Context,
	callOpts *bind.CallOpts,
	fastConfirmCallData []byte,
	safeTxHash [32]byte,
) (bool, error) {
	var signatures []byte
	approvedHashCount := uint64(0)
	for _, owner := range f.owners {
		var approved *big.Int
		// No need check if wallet has approved the hash,
		// since checkApprovedHashAndExecTransaction is called only after wallet has approved the hash.
		if f.assertionChain.StakerAddress() == owner {
			approved = common.Big1
		} else {
			var err error
			approved, err = retry.UntilSucceeds(callOpts.Context, func() (*big.Int, error) {
				return f.safe.ApprovedHashes(callOpts, owner, safeTxHash)
			})
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
		receipt, err := f.assertionChain.transact(ctx, f.assertionChain.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return f.safe.ExecTransaction(
				opts,
				f.assertionChain.RollupAddress(),
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
		})
		if err != nil {
			return false, err
		}
		if len(receipt.Logs) == 0 {
			return false, errors.New("no logs observed from hash approval")
		}
		return true, nil
	}
	log.Info("Not enough Safe tx approvals yet to fast confirm", "safeHash", common.BytesToHash(safeTxHash[:]).Hex())
	return false, nil
}
