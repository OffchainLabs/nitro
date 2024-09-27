// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestArbFunctionTable(t *testing.T) {
	t.Parallel()

	evm := newMockEVMForTesting()
	ftab := ArbFunctionTable{}
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// should be a noop
	err := ftab.Upload(context, evm, []byte{0, 0, 0, 0})
	Require(t, err)

	size, err := ftab.Size(context, evm, addr)
	Require(t, err)
	if size.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Size should be 0")
	}

	_, _, _, err = ftab.Get(context, evm, addr, big.NewInt(10))
	if err == nil {
		t.Fatal("Should error")
	}
}
