package arbutil

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSlotAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		args [][]byte
		want []byte
	}{
		{
			name: "isBatchPoster[batchPosterAddr]", // Keccak256(addr, 3)
			args: [][]byte{
				common.FromHex("0xC1b634853Cb333D3aD8663715b08f41A3Aec47cc"), // batch poster address
				{3},
			},
			want: common.HexToHash("0xa10aa54071443520884ed767b0684edf43acec528b7da83ab38ce60126562660").Bytes(),
		},
		{
			name: "allowedContracts[msg.sender]", // Keccak256(msg.sender, 1)
			args: [][]byte{
				common.FromHex("0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6"), // sequencer address
				{1},
			},
			want: common.HexToHash("0xe85fd79f89ff278fc57d40aecb7947873df9f0beac531c8f71a98f630e1eab62").Bytes(),
		},
		{
			name: "allowedRefundees[refundee]", // Keccak256(msg.sender, 2)
			args: [][]byte{
				common.FromHex("0xC1b634853Cb333D3aD8663715b08f41A3Aec47cc"), // batch poster address
				{2},
			},
			want: common.HexToHash("0x7686888b19bb7b75e46bb1aa328b65150743f4899443d722f0adf8e252ccda41").Bytes(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := slotAddress(tc.args...)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("slotAddress(%x) = %x, want %x", tc.args, got, tc.want)
			}
		})
	}

}
