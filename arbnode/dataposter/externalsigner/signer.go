// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package externalsigner

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
)

// Signer function callback when a contract requires a method to sign
// the transaction before submission.
//
// This can be local or external, hence the context parameter.
type SignerFn func(context.Context, common.Address, *types.Transaction) (*types.Transaction, error)

// External signer function with ethereum address of the signer.
type ExternalSigner struct {
	Sender common.Address
	Signer SignerFn
}

// Returns signer function and ethereum address of the signer.
//
// Returns an error if address isn't specified or if it can't connect to the
// signer RPC server.
func NewExternalSigner(ctx context.Context, opts *config.ExternalSignerCfg) (*ExternalSigner, error) {
	if opts.Address == "" {
		return nil, errors.New("external signer (From) address specified")
	}

	client, err := rpcClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error connecting external signer: %w", err)
	}
	sender := common.HexToAddress(opts.Address)

	signer := func(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		// According to the "eth_signTransaction" API definition, this should be
		// RLP encoded transaction object.
		// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_signtransaction
		var data hexutil.Bytes
		args, err := txToSignTxArgs(addr, tx)
		if err != nil {
			return nil, fmt.Errorf("error converting transaction to sendTxArgs: %w", err)
		}
		if err := client.CallContext(ctx, &data, opts.Method, args); err != nil {
			return nil, fmt.Errorf("making signing request to external signer: %w", err)
		}
		signedTx := &types.Transaction{}
		if err := signedTx.UnmarshalBinary(data); err != nil {
			return nil, fmt.Errorf("unmarshaling signed transaction: %w", err)
		}
		hasher := types.LatestSignerForChainID(tx.ChainId())
		gotTx, err := args.ToTransaction()
		if err != nil {
			return nil, fmt.Errorf("converting transaction arguments into transaction: %w", err)
		}
		if h := hasher.Hash(gotTx); h != hasher.Hash(signedTx) {
			return nil, fmt.Errorf("transaction: %x from external signer differs from request: %x", hasher.Hash(signedTx), h)
		}
		// Ensure the returned transaction is signed by the expected address.
		// Use the hasher derived from the signed transaction's chain ID to
		// correctly recover the sender address regardless of the input tx fields.
		recoveryHasher := types.LatestSignerForChainID(signedTx.ChainId())
		from, err := types.Sender(recoveryHasher, signedTx)
		if err != nil {
			return nil, fmt.Errorf("recovering signer address: %w", err)
		}
		if from != sender {
			return nil, fmt.Errorf("external signer returned tx from %s, expected %s", from.Hex(), sender.Hex())
		}
		return signedTx, nil
	}

	return &ExternalSigner{
		Sender: sender,
		Signer: signer,
	}, nil
}

func (s *ExternalSigner) TxOpts() *bind.TransactOpts {
	return &bind.TransactOpts{
		From: s.Sender,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return s.Signer(context.TODO(), address, tx)
		},
	}
}
