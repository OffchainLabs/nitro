package externalsigner

import (
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

func (a *SignTxArgs) ToTransaction() *types.Transaction {
	if !a.isEIP4844() {
		return a.SendTxArgs.ToTransaction()
	}
	to := common.Address{}
	if a.To != nil {
		to = a.To.Address()
	}
	var input []byte
	if a.Input != nil {
		input = *a.Input
	} else if a.Data != nil {
		input = *a.Data
	}
	al := types.AccessList{}
	if a.AccessList != nil {
		al = *a.AccessList
	}
	return types.NewTx(&types.BlobTx{
		To:         to,
		Nonce:      uint64(a.SendTxArgs.Nonce),
		Gas:        uint64(a.Gas),
		GasFeeCap:  uint256.NewInt(a.MaxFeePerGas.ToInt().Uint64()),
		GasTipCap:  uint256.NewInt(a.MaxPriorityFeePerGas.ToInt().Uint64()),
		Value:      uint256.NewInt(a.Value.ToInt().Uint64()),
		Data:       input,
		AccessList: al,
		BlobFeeCap: uint256.NewInt(a.BlobFeeCap.ToInt().Uint64()),
		BlobHashes: a.BlobHashes,
		Sidecar: &types.BlobTxSidecar{
			Blobs:       a.Blobs,
			Commitments: a.Commitments,
			Proofs:      a.Proofs,
		},
		ChainID: uint256.NewInt(a.ChainID.ToInt().Uint64()),
	})
}

func (a *SignTxArgs) isEIP4844() bool {
	return a.BlobHashes != nil || a.BlobFeeCap != nil
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
