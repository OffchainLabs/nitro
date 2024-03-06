package externalsigner

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

var (
	blobTx = types.NewTx(
		&types.BlobTx{
			ChainID:   uint256.NewInt(1337),
			Nonce:     13,
			GasTipCap: uint256.NewInt(1),
			GasFeeCap: uint256.NewInt(1),
			Gas:       3,
			To:        common.Address{},
			Value:     uint256.NewInt(1),
			Data:      []byte{0x01, 0x02, 0x03},
			BlobHashes: []common.Hash{
				common.BigToHash(big.NewInt(1)),
				common.BigToHash(big.NewInt(2)),
				common.BigToHash(big.NewInt(3)),
			},
			Sidecar: &types.BlobTxSidecar{},
		},
	)
	dynamicFeeTx = types.NewTx(
		&types.DynamicFeeTx{
			Nonce:     13,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(1),
			Gas:       3,
			To:        nil,
			Value:     big.NewInt(1),
			Data:      []byte{0x01, 0x02, 0x03},
		},
	)
)

// TestToTranssaction tests that tranasction converted to SignTxArgs and then
// back to Transaction results in the same hash.
func TestToTranssaction(t *testing.T) {
	for _, tc := range []struct {
		desc string
		tx   *types.Transaction
	}{
		{
			desc: "blob transaction",
			tx:   blobTx,
		},
		{
			desc: "dynamic fee transaction",
			tx:   dynamicFeeTx,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			signTxArgs, err := TxToSignTxArgs(common.Address{}, tc.tx)
			if err != nil {
				t.Fatalf("TxToSignTxArgs() unexpected error: %v", err)
			}
			got := signTxArgs.ToTransaction()
			hasher := types.LatestSignerForChainID(tc.tx.ChainId())
			if h, g := hasher.Hash(tc.tx), hasher.Hash(got); h != g {
				t.Errorf("ToTransaction() got hash: %v want: %v", g, h)
			}
		})
	}

}
