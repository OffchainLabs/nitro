// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package smoketest

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var p256VerifyTests = []systest.Scenario{
	systest.Test(testRunP256Verify, systest.MatrixArbOS(params.ArbosVersion_20, params.ArbosVersion_30)),
}

func testRunP256Verify(env *systest.Env) {
	addr := common.BytesToAddress([]byte{0x01, 0x00})
	got := env.L2.CallContract(ethereum.CallMsg{
		From:  env.L2.Info.GetAddress("Owner"),
		To:    &addr,
		Gas:   env.L2.Info.TransferGas,
		Data:  common.Hex2Bytes("4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e"),
		Value: big.NewInt(1e12),
	}, nil)
	want := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
	if env.Spec.ArbOSVersion.UnwrapOr(0) < params.ArbosVersion_30 {
		want = nil
	}
	env.True(bytes.Equal(want, got), "P256Verify() = %x, want %x", got, want)
}
