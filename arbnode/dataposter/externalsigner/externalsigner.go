package externalsigner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/holiman/uint256"
)

type SignTxArgs struct {
	*apitypes.SendTxArgs

	BlobFeeCap        *uint256.Int         `json:"blobFeeCap,omitempty"`
	BlobVersionHashes []common.Hash        `json:"blobVersionedHashes,omitempty"`
	Sidecar           *types.BlobTxSidecar `json:"sidecar,omitempty" rlp:"-"`
}

func (a *SignTxArgs) ToTransaction() *types.Transaction {
	// Sidecar field must be set when BlobTx is used to create a transction
	// for signing.
	if a.Sidecar == nil {
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
		BlobFeeCap: a.BlobFeeCap,
		BlobHashes: a.BlobVersionHashes,
		Sidecar:    a.Sidecar,
	})
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
	var blobFeeCap *uint256.Int
	if tx.BlobGasFeeCap() != nil {
		blobFeeCap = uint256.NewInt(tx.BlobGasFeeCap().Uint64())
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
		BlobFeeCap:        blobFeeCap,
		BlobVersionHashes: tx.BlobHashes(),
		Sidecar:           tx.BlobTxSidecar(),
	}, nil
}
