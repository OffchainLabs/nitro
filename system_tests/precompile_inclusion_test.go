// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/execution/gethexec"
)

var (
	// Homestead
	ecrecover = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0x1}),
		input:    common.Hex2Bytes("38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02"),
		expected: common.Hex2Bytes("000000000000000000000000ceaccac640adf55b2028469bd36ba501f28b699d"),
	}
	// Byzantium
	bn256AddByzantium = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0x6}),
		input:    common.Hex2Bytes("18b18acfb4c2c30276db5411368e7185b311dd124691610c5d3b74034e093dc9063c909c4720840cb5134cb9f59fa749755796819658d32efc0d288198f3726607c2b7f58a84bd6145f00c9c2bc0bb1a187f20ff2c92963a88019e7c6a014eed06614e20c147e940f2d70da3f74c9a17df361706a4485c742bd6788478fa17d7"),
		expected: common.Hex2Bytes("2243525c5efd4b9c3d3c45ac0ca3fe4dd85e830a4ce6b65fa1eeaee202839703301d1d33be6da8e509df21cc35964723180eed7532537db9ae5e7d48f195c915"),
	}
	// Istanbul / Berlin
	blake2F = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0x9}),
		input:    common.Hex2Bytes("0000000048c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001"),
		expected: common.Hex2Bytes("08c9bcf367e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d282e6ad7f520e511f6c3e2b8c68059b9442be0454267ce079217e1319cde05b"),
	}
	// Cancun
	kzgPointEvaluation = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0xa}),
		input:    common.Hex2Bytes("01e798154708fe7789429634053cbf9f99b619f9f084048927333fce637f549b564c0a11a0f704f4fc3e8acfe0f8245f0ad1347b378fbf96e206da11a5d3630624d25032e67a7e6a4910df5834b8fe70e6bcfeeac0352434196bdf4b2485d5a18f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7873033e038326e87ed3e1276fd140253fa08e9fc25fb2d9a98527fc22a2c9612fbeafdad446cbc7bcdbdcd780af2c16a"),
		expected: common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000100073eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"),
	}
	// EIP-7212
	p256Verify = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0x01, 0x00}),
		input:    common.Hex2Bytes("bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca6050232ba3a8be6b94d5ec80a6d9d1190a436effe50d85a1eee859b8cc6af9bd5c2e184cd60b855d442f5b3c7b11eb6c4e0ae7525fe710fab9aa7c77a67f79e6fadd762927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e"),
		expected: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
	}
	// Prague
	bls12381G1Add = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0xb}),
		input:    bytes.Repeat([]byte{0}, 256),
		expected: bytes.Repeat([]byte{0}, 128),
	}
	bls12381G1MultiExp = precompileCaseProvider{
		addr:     common.BytesToAddress([]byte{0xc}),
		input:    bytes.Repeat([]byte{0}, 320),
		expected: bytes.Repeat([]byte{0}, 128),
	}
)

func TestVersion11(t *testing.T) {
	testPrecompiles(t, params.ArbosVersion_11, ecrecover.Included(), bn256AddByzantium.Included(), blake2F.Included(), kzgPointEvaluation.NotIncluded(), p256Verify.NotIncluded(), bls12381G1Add.NotIncluded(), bls12381G1MultiExp.NotIncluded())
}

func TestVersion30(t *testing.T) {
	testPrecompiles(t, params.ArbosVersion_30, ecrecover.Included(), bn256AddByzantium.Included(), kzgPointEvaluation.Included(), p256Verify.Included(), bls12381G1Add.NotIncluded(), bls12381G1MultiExp.NotIncluded())
}

func TestVersion40(t *testing.T) {
	testPrecompiles(t, params.ArbosVersion_40, bn256AddByzantium.Included(), kzgPointEvaluation.Included(), p256Verify.Included(), bls12381G1Add.NotIncluded(), bls12381G1MultiExp.NotIncluded())
}

func TestArbOSVersion50(t *testing.T) {
	testPrecompiles(t, params.ArbosVersion_50, kzgPointEvaluation.Included(), bls12381G1Add.Included(), bls12381G1MultiExp.Included())
}

func TestArbOSVersion60(t *testing.T) {
	testPrecompiles(t, params.ArbosVersion_60, kzgPointEvaluation.Included(), bls12381G1Add.Included(), bls12381G1MultiExp.Included())
}

func testPrecompiles(t *testing.T, arbosVersion uint64, cases ...precompileCase) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(arbosVersion)
	builder.execConfig.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessLikelyCompatible
	cleanup := builder.Build(t)
	defer cleanup()
	for _, c := range cases {
		res, err := builder.L2.Client.CallContract(context.Background(), ethereum.CallMsg{To: &c.addr, Data: c.in}, nil)
		Require(t, err)
		if !bytes.Equal(res, c.out) {
			t.Errorf("Expected %v [%d], got %v [%d]", c.out, len(c.out), res, len(res))
		}
	}

}

type precompileCase struct {
	addr common.Address
	in   []byte
	out  []byte
}

type precompileCaseProvider struct {
	addr     common.Address
	input    []byte
	expected []byte
}

func (c precompileCaseProvider) Included() precompileCase {
	return precompileCase{
		addr: c.addr,
		in:   c.input,
		out:  c.expected,
	}
}

func (c precompileCaseProvider) NotIncluded() precompileCase {
	return precompileCase{
		addr: c.addr,
		in:   c.input,
		out:  []byte{},
	}
}
