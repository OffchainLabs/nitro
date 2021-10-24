//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"math/big"
	"testing"
)

func TestEvents(t *testing.T) {

	// create a minimal evm that supports just enough to create logs
	evm := vm.EVM{
		StateDB: storage.NewMemoryBackedStateDB(),
		Context: vm.BlockContext{
			BlockNumber: big.NewInt(1024),
			GasLimit:    ^uint64(0),
		},
	}

	debugContractAddr := common.HexToAddress("ff")
	contract := Precompiles()[debugContractAddr]

	var method PrecompileMethod

	for _, available := range contract.Precompile().methods {
		if available.name == "Events" {
			method = available
			break
		}
	}

	var data []byte
	payload := [][]byte{
		method.template.ID,             // select the `Events` method
		common.Hash{}.Bytes(),          // set the flag to false
		crypto.Keccak256([]byte{0x00}), // set the value to something known
	}
	for _, bytes := range payload {
		data = append(data, bytes...)
	}

	caller := common.HexToAddress("aaaaaaaabbbbbbbbccccccccdddddddd")
	number := big.NewInt(0x9364)

	output, err := contract.Call(
		data,
		debugContractAddr,
		debugContractAddr,
		caller,
		number,
		false,
		&evm,
	)
	check(t, err, "call failed")

	outputAddr := common.BytesToAddress(output[:32])
	outputData := new(big.Int).SetBytes(output[32:])

	if outputAddr != caller {
		t.Fatal("unexpected output address", outputAddr, "instead of", caller)
	}
	if outputData.Cmp(number) != 0 {
		t.Fatal("unexpected output number", outputData, "instead of", number)
	}
}

func check(t *testing.T, err error, str ...string) {
	if err != nil {
		t.Fatal(err, str)
	}
}
