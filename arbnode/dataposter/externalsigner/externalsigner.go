package externalsigner

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/holiman/uint256"
)

type SignTxArgs struct {
	*apitypes.SendTxArgs

	// Feilds for BlobTx type transactions.
	BlobFeeCap *hexutil.Big  `json:"maxFeePerBlobGas"`
	BlobHashes []common.Hash `json:"blobVersionedHashes,omitempty"`

	// Blob sidecar fields for BlobTx type transactions.
	// These are optional if BlobHashes are already present, since these
	// are not included in the hash/signature.
	Blobs       []kzg4844.Blob       `json:"blobs"`
	Commitments []kzg4844.Commitment `json:"commitments"`
	Proofs      []kzg4844.Proof      `json:"proofs"`
}

// data retrieves the transaction calldata. Input field is preferred.
func (args *SignTxArgs) data() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}

// ToTransaction converts the arguments to a transaction.
func (args *SignTxArgs) ToTransaction() (*types.Transaction, error) {
	// Add the To-field, if specified
	var to *common.Address
	if args.To != nil {
		dstAddr := args.To.Address()
		to = &dstAddr
	}
	if err := args.validateTxSidecar(); err != nil {
		return nil, err
	}
	var data types.TxData
	switch {
	case args.BlobHashes != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.BlobTx{
			To:         *to,
			ChainID:    uint256.MustFromBig((*big.Int)(args.ChainID)),
			Nonce:      uint64(args.Nonce),
			Gas:        uint64(args.Gas),
			GasFeeCap:  uint256.MustFromBig((*big.Int)(args.MaxFeePerGas)),
			GasTipCap:  uint256.MustFromBig((*big.Int)(args.MaxPriorityFeePerGas)),
			Value:      uint256.MustFromBig((*big.Int)(&args.Value)),
			Data:       args.data(),
			AccessList: al,
			BlobHashes: args.BlobHashes,
			BlobFeeCap: uint256.MustFromBig((*big.Int)(args.BlobFeeCap)),
		}
		if args.Blobs != nil {
			data.(*types.BlobTx).Sidecar = &types.BlobTxSidecar{
				Blobs:       args.Blobs,
				Commitments: args.Commitments,
				Proofs:      args.Proofs,
			}
		}

	case args.MaxFeePerGas != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.DynamicFeeTx{
			To:         to,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(args.Nonce),
			Gas:        uint64(args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(&args.Value),
			Data:       args.data(),
			AccessList: al,
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			To:         to,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(args.Nonce),
			Gas:        uint64(args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(&args.Value),
			Data:       args.data(),
			AccessList: *args.AccessList,
		}
	default:
		data = &types.LegacyTx{
			To:       to,
			Nonce:    uint64(args.Nonce),
			Gas:      uint64(args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(&args.Value),
			Data:     args.data(),
		}
	}

	return types.NewTx(data), nil
}

// validateTxSidecar validates blob data, if present
func (args *SignTxArgs) validateTxSidecar() error {
	// No blobs, we're done.
	if args.Blobs == nil {
		return nil
	}

	n := len(args.Blobs)
	// Assume user provides either only blobs (w/o hashes), or
	// blobs together with commitments and proofs.
	if args.Commitments == nil && args.Proofs != nil {
		return errors.New(`blob proofs provided while commitments were not`)
	} else if args.Commitments != nil && args.Proofs == nil {
		return errors.New(`blob commitments provided while proofs were not`)
	}

	// len(blobs) == len(commitments) == len(proofs) == len(hashes)
	if args.Commitments != nil && len(args.Commitments) != n {
		return fmt.Errorf("number of blobs and commitments mismatch (have=%d, want=%d)", len(args.Commitments), n)
	}
	if args.Proofs != nil && len(args.Proofs) != n {
		return fmt.Errorf("number of blobs and proofs mismatch (have=%d, want=%d)", len(args.Proofs), n)
	}
	if args.BlobHashes != nil && len(args.BlobHashes) != n {
		return fmt.Errorf("number of blobs and hashes mismatch (have=%d, want=%d)", len(args.BlobHashes), n)
	}

	if args.Commitments == nil {
		// Generate commitment and proof.
		commitments := make([]kzg4844.Commitment, n)
		proofs := make([]kzg4844.Proof, n)
		for i, b := range args.Blobs {
			c, err := kzg4844.BlobToCommitment(b)
			if err != nil {
				return fmt.Errorf("blobs[%d]: error computing commitment: %w", i, err)
			}
			commitments[i] = c
			p, err := kzg4844.ComputeBlobProof(b, c)
			if err != nil {
				return fmt.Errorf("blobs[%d]: error computing proof: %w", i, err)
			}
			proofs[i] = p
		}
		args.Commitments = commitments
		args.Proofs = proofs
	} else {
		for i, b := range args.Blobs {
			b := b // avoid memeroy aliasing
			if err := kzg4844.VerifyBlobProof(b, args.Commitments[i], args.Proofs[i]); err != nil {
				return fmt.Errorf("failed to verify blob proof: %w", err)
			}
		}
	}

	hashes := make([]common.Hash, n)
	hasher := sha256.New()
	for i, c := range args.Commitments {
		c := c // avoid memeroy aliasing
		hashes[i] = kzg4844.CalcBlobHashV1(hasher, &c)
	}
	if args.BlobHashes != nil {
		for i, h := range hashes {
			if h != args.BlobHashes[i] {
				return fmt.Errorf("blob hash verification failed (have=%s, want=%s)", args.BlobHashes[i], h)
			}
		}
	} else {
		args.BlobHashes = hashes
	}
	return nil
}

// TxToSignTxArgs converts transaction to SendTxArgs. This is needed for
// external signer to specify From field.
func TxToSignTxArgs(addr common.Address, tx *types.Transaction) (*SignTxArgs, error) {
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
	)
	if tx.BlobTxSidecar() != nil {
		blobs = tx.BlobTxSidecar().Blobs
		commitments = tx.BlobTxSidecar().Commitments
		proofs = tx.BlobTxSidecar().Proofs
	}
	return &SignTxArgs{
		SendTxArgs: &apitypes.SendTxArgs{
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
		},
		BlobFeeCap:  (*hexutil.Big)(tx.BlobGasFeeCap()),
		BlobHashes:  tx.BlobHashes(),
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}, nil
}
