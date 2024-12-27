package message

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestTimeboostedInDifferentScenarios(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		blockMetadata common.BlockMetadata
		txs           []bool // Array representing whether the tx is timeboosted or not. First tx is always false as its an arbitrum internal tx
	}{
		{
			name:          "block has no timeboosted tx",
			blockMetadata: []byte{0, 0, 0},                                         // 00000000 00000000
			txs:           []bool{false, false, false, false, false, false, false}, // num of tx in this block = 7
		},
		{
			name:          "block has only one timeboosted tx",
			blockMetadata: []byte{0, 2},        // 00000000 01000000
			txs:           []bool{false, true}, // num of tx in this block = 2
		},
		{
			name:          "block has multiple timeboosted tx",
			blockMetadata: []byte{0, 86, 145},                                                                                              // 00000000 01101010 10001001
			txs:           []bool{false, true, true, false, true, false, true, false, true, false, false, false, true, false, false, true}, // num of tx in this block = 16
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for txIndex, want := range tc.txs {
				have, err := tc.blockMetadata.IsTxTimeboosted(txIndex)
				if err != nil {
					t.Fatalf("error getting timeboosted bit for tx of index %d: %v", txIndex, err)
				}
				if want != have {
					t.Fatalf("incorrect timeboosted bit for tx of index %d, Got: %v, Want: %v", txIndex, have, want)
				}
			}
		})
	}
}
