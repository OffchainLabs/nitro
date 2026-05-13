// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dataposter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
)

// signerFn is a signer function callback when a contract requires a method to
// sign the transaction before submission.
// This can be local or external, hence the context parameter.
type signerFn func(context.Context, common.Address, *types.Transaction) (*types.Transaction, error)

func rpcClient(ctx context.Context, opts *config.ExternalSignerCfg) (*rpc.Client, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Dataposter verifies that signed transaction was signed by the account
		// that it expects to be signed with. So signer is already authenticated
		// on application level and does not need to rely on TLS for authentication.
		InsecureSkipVerify: opts.InsecureSkipVerify, // #nosec G402
	}

	if opts.ClientCert != "" && opts.ClientPrivateKey != "" {
		log.Info("Client certificate for external signer is enabled")
		clientCert, err := tls.LoadX509KeyPair(opts.ClientCert, opts.ClientPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error loading client certificate and private key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{clientCert}
	}

	if opts.RootCA != "" {
		rootCrt, err := os.ReadFile(opts.RootCA)
		if err != nil {
			return nil, fmt.Errorf("error reading external signer root CA: %w", err)
		}
		rootCertPool := x509.NewCertPool()
		rootCertPool.AppendCertsFromPEM(rootCrt)
		tlsCfg.RootCAs = rootCertPool
	}

	return rpc.DialOptions(
		ctx,
		opts.URL,
		rpc.WithHTTPClient(
			&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsCfg,
				},
			},
		),
	)
}

// TxToSignTxArgs converts transaction to SendTxArgs. This is needed for
// external signer to specify From field.
func TxToSignTxArgs(addr common.Address, tx *types.Transaction) (*apitypes.SendTxArgs, error) {
	var to *common.MixedcaseAddress
	if tx.To() != nil {
		to = new(common.MixedcaseAddress)
		*to = common.NewMixedcaseAddress(*tx.To())
	}
	data := (hexutil.Bytes)(tx.Data())
	val := (*hexutil.Big)(tx.Value())
	if val == nil {
		val = (*hexutil.Big)(big.NewInt(0))
	}
	al := tx.AccessList()
	var (
		blobs       []kzg4844.Blob
		commitments []kzg4844.Commitment
		proofs      []kzg4844.Proof
		blobVersion byte
	)
	if tx.BlobTxSidecar() != nil {
		blobs = tx.BlobTxSidecar().Blobs
		commitments = tx.BlobTxSidecar().Commitments
		proofs = tx.BlobTxSidecar().Proofs
		blobVersion = tx.BlobTxSidecar().Version
	}
	return &apitypes.SendTxArgs{
		From:                 common.NewMixedcaseAddress(addr),
		To:                   to,
		Gas:                  hexutil.Uint64(tx.Gas()),
		GasPrice:             (*hexutil.Big)(tx.GasPrice()),
		MaxFeePerGas:         (*hexutil.Big)(tx.GasFeeCap()),
		MaxPriorityFeePerGas: (*hexutil.Big)(tx.GasTipCap()),
		Value:                *val,
		Nonce:                hexutil.Uint64(tx.Nonce()),
		Data:                 &data,
		AccessList:           &al,
		ChainID:              (*hexutil.Big)(tx.ChainId()),
		BlobFeeCap:           (*hexutil.Big)(tx.BlobGasFeeCap()),
		BlobHashes:           tx.BlobHashes(),
		BlobVersion:          blobVersion,
		Blobs:                blobs,
		Commitments:          commitments,
		Proofs:               proofs,
	}, nil
}

func ExternalSignerTxOpts(ctx context.Context, opts *config.ExternalSignerCfg) (*bind.TransactOpts, error) {
	signer, sender, err := externalSigner(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &bind.TransactOpts{
		From: sender,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return signer(context.TODO(), address, tx)
		},
	}, nil
}

// externalSigner returns signer function and ethereum address of the signer.
// Returns an error if address isn't specified or if it can't connect to the
// signer RPC server.
func externalSigner(ctx context.Context, opts *config.ExternalSignerCfg) (signerFn, common.Address, error) {
	if opts.Address == "" {
		return nil, common.Address{}, errors.New("external signer (From) address specified")
	}

	client, err := rpcClient(ctx, opts)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("error connecting external signer: %w", err)
	}
	sender := common.HexToAddress(opts.Address)
	return func(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		// According to the "eth_signTransaction" API definition, this should be
		// RLP encoded transaction object.
		// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_signtransaction
		var data hexutil.Bytes
		args, err := TxToSignTxArgs(addr, tx)
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
	}, sender, nil
}
