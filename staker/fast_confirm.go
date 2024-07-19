package staker

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
	gasRefunder               common.Address
	l1Reader                  *headerreader.HeaderReader
}

func NewFastConfirmSafe(
	callOpts bind.CallOpts,
	fastConfirmSafeAddress common.Address,
	builder *txbuilder.Builder,
	wallet ValidatorWalletInterface,
	gasRefunder common.Address,
	l1Reader *headerreader.HeaderReader,
) (*FastConfirmSafe, error) {
	fastConfirmSafe := &FastConfirmSafe{
		builder:     builder,
		wallet:      wallet,
		gasRefunder: gasRefunder,
		l1Reader:    l1Reader,
	}
	safe, err := contractsgen.NewSafe(fastConfirmSafeAddress, builder)
	if err != nil {
		return nil, err
	}
	fastConfirmSafe.safe = safe
	owners, err := safe.GetOwners(&callOpts)
	if err != nil {
		return nil, err
	}

	// This is needed because safe contract needs owners to be sorted.
	sort.Slice(owners, func(i, j int) bool {
		return owners[i].Cmp(owners[j]) < 0
	})
	fastConfirmSafe.owners = owners
	threshold, err := safe.GetThreshold(&callOpts)
	if err != nil {
		return nil, err
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

func (f *FastConfirmSafe) tryFastConfirmation(ctx context.Context, blockHash common.Hash, sendRoot common.Hash) error {
	fastConfirmCallData, err := f.createFastConfirmCalldata(blockHash, sendRoot)
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
	auth, err := f.builder.Auth(ctx)
	if err != nil {
		return err
	}
	_, err = f.safe.ApproveHash(auth, safeTxHash)
	if err != nil {
		return err
	}
	if !f.wallet.CanBatchTxs() {
		err = f.flushTransactions(ctx)
		if err != nil {
			return err
		}
	}
	return f.checkApprovedHashAndExecTransaction(ctx, fastConfirmCallData, safeTxHash)
}

func (f *FastConfirmSafe) flushTransactions(ctx context.Context) error {
	arbTx, err := f.wallet.ExecuteTransactions(ctx, f.builder, f.gasRefunder)
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
	f.builder.ClearTransactions()
	return nil
}

func (f *FastConfirmSafe) createFastConfirmCalldata(
	blockHash common.Hash, sendRoot common.Hash,
) ([]byte, error) {
	calldata, err := f.fastConfirmNextNodeMethod.Inputs.Pack(
		blockHash,
		sendRoot,
	)
	if err != nil {
		return nil, err
	}
	fullCalldata := append([]byte{}, f.fastConfirmNextNodeMethod.ID...)
	fullCalldata = append(fullCalldata, calldata...)
	return fullCalldata, nil
}

func (f *FastConfirmSafe) checkApprovedHashAndExecTransaction(ctx context.Context, fastConfirmCallData []byte, safeTxHash [32]byte) error {
	var signatures []byte
	approvedHashCount := uint64(0)
	for _, owner := range f.owners {
		if f.wallet.Address() == nil {
			return errors.New("wallet address is nil")
		}
		var approved *big.Int
		// No need check if wallet has approved the hash,
		// since checkApprovedHashAndExecTransaction is called only after wallet has approved the hash.
		if *f.wallet.Address() == owner {
			approved = common.Big1
		} else {
			var err error
			approved, err = f.safe.ApprovedHashes(&bind.CallOpts{Context: ctx}, owner, safeTxHash)
			if err != nil {
				return err
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
		auth, err := f.builder.Auth(ctx)
		if err != nil {
			return err
		}
		_, err = f.safe.ExecTransaction(
			auth,
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
			return err
		}
	}
	return nil
}
