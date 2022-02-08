//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompile_fuzz

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/precompiles"
)

const fuzzGas uint64 = 1200000

var evm *vm.EVM

func init() {
	arbstate.RequireHookedGeth()

	// Create a StateDB
	sdb, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		panic(err)
	}
	_, err = arbosState.InitializeArbosState(sdb, burn.NewSystemBurner(false))
	if err != nil {
		panic(err)
	}

	// Create an EVM
	txContext := vm.TxContext{
		GasPrice: new(big.Int),
	}
	blockContext := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     nil,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int),
		Time:        new(big.Int),
		Difficulty:  new(big.Int),
		GasLimit:    fuzzGas,
		BaseFee:     new(big.Int),
	}
	evm = vm.NewEVM(blockContext, txContext, sdb, params.ArbitrumTestChainConfig(), vm.Config{})
}

func Fuzz(input []byte) int {
	// We require at least two bytes: one for the address selection and the next for the method selection
	if len(input) < 2 {
		return 0
	}

	// Pick a precompile address based on the first byte of the input
	var addr common.Address
	addr[19] = input[0]
	input = input[1:]

	// Pick a precompile method based on the second byte of the input
	if precompile := precompiles.Precompiles()[addr]; precompile != nil {
		sigs := precompile.Precompile().Get4ByteMethodSignatures()
		if int(input[0]) < len(sigs) {
			input = append(sigs[input[0]][:], input[0:]...)
		}
	}

	// Create and apply a message
	msg := types.NewMessage(
		common.Address{},
		&addr,
		0,
		new(big.Int),
		fuzzGas,
		new(big.Int),
		new(big.Int),
		new(big.Int),
		input,
		nil,
		true,
	)
	gp := core.GasPool(fuzzGas)
	_, _ = core.ApplyMessage(evm, msg, &gp)

	return 0
}
