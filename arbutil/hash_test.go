package arbutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
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
				common.FromHex("0xC1b634853Cb333D3aD8663715b08f41A3Aec47cc"), // mainnet batch poster address
				{3},
			},
			want: common.HexToHash("0xa10aa54071443520884ed767b0684edf43acec528b7da83ab38ce60126562660").Bytes(),
		},
		{
			name: "allowedContracts[msg.sender]", // Keccak256(msg.sender, 1)
			args: [][]byte{
				common.FromHex("0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6"), // mainnet sequencer address
				{1},
			},
			want: common.HexToHash("0xe85fd79f89ff278fc57d40aecb7947873df9f0beac531c8f71a98f630e1eab62").Bytes(),
		},
		{
			name: "allowedRefundees[refundee]", // Keccak256(msg.sender, 2)
			args: [][]byte{
				common.FromHex("0xC1b634853Cb333D3aD8663715b08f41A3Aec47cc"), // mainnet batch poster address
				{2},
			},
			want: common.HexToHash("0x7686888b19bb7b75e46bb1aa328b65150743f4899443d722f0adf8e252ccda41").Bytes(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := PaddedKeccak256(tc.args...)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("slotAddress(%x) = %x, want %x", tc.args, got, tc.want)
			}
		})
	}

}

func TestSumBytes(t *testing.T) {
	for _, tc := range []struct {
		desc       string
		a, b, want []byte
	}{
		{
			desc: "simple case",
			a:    []byte{0x0a, 0x0b},
			b:    []byte{0x03, 0x04},
			want: common.HexToHash("0x0d0f").Bytes(),
		},
		{
			desc: "carry over last byte",
			a:    []byte{0x0a, 0xff},
			b:    []byte{0x01},
			want: common.HexToHash("0x0b00").Bytes(),
		},
		{
			desc: "overflow",
			a:    common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff").Bytes(),
			b:    []byte{0x01},
			want: common.HexToHash("0x00").Bytes(),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := SumBytes(tc.a, tc.b)
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("SumBytes(%x, %x) = %x want: %x", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestBrutforce(t *testing.T) {
	M := map[common.Hash]bool{
		common.HexToHash("0xa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c688"): true,
		common.HexToHash("0xf652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d40"): true,
		common.HexToHash("0xf652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f"): true,
		common.HexToHash("0xa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c689"): true,
	}

	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			addr := SumBytes(PaddedKeccak256(intToBytes(t, i)), intToBytes(t, j))
			if M[common.BytesToHash(addr)] {
				t.Errorf("anodar yes, i: %v, j: %v, \taddr: %x", i, j, addr)
			}
		}

	}
}

func intToBytes(t *testing.T, val int) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, int64(val))
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}
