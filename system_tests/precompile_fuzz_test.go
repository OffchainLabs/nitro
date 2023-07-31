// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/precompiles"
)

const fuzzGas uint64 = 1200000

func FuzzPrecompiles(f *testing.F) {
	gethhook.RequireHookedGeth()

	f.Fuzz(func(t *testing.T, precompileSelector byte, methodSelector byte, input []byte) {
		// Create a StateDB
		sdb, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		if err != nil {
			panic(err)
		}
		burner := burn.NewSystemBurner(nil, false)
		chainConfig := params.ArbitrumDevTestChainConfig()
		_, err = arbosState.InitializeArbosState(sdb, burner, chainConfig, arbostypes.TestInitMessage)
		if err != nil {
			panic(err)
		}

		// Create an EVM
		gp := core.GasPool{}
		gp.AddGas(fuzzGas)
		gp.AddDataGas(fuzzGas)
		txContext := vm.TxContext{
			GasPrice: common.Big1,
		}
		blockContext := vm.BlockContext{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     nil,
			Coinbase:    common.Address{},
			BlockNumber: new(big.Int),
			Time:        0,
			Difficulty:  new(big.Int),
			GasLimit:    fuzzGas,
			BaseFee:     common.Big1,
		}
		evm := vm.NewEVM(blockContext, txContext, sdb, chainConfig, vm.Config{})

		// Pick a precompile address based on the first byte of the input
		var addr common.Address
		addr[19] = precompileSelector

		// Pick a precompile method based on the second byte of the input
		if precompile := precompiles.Precompiles()[addr]; precompile != nil {
			sigs := precompile.Precompile().Get4ByteMethodSignatures()
			if int(methodSelector) < len(sigs) {
				newInput := make([]byte, 4)
				copy(newInput, sigs[methodSelector][:])
				newInput = append(newInput, input...)
				input = newInput
			}
		}

		// Create and apply a message
		msg := &core.Message{
			From:       common.Address{},
			To:         &addr,
			Nonce:      0,
			Value:      new(big.Int),
			GasLimit:   fuzzGas,
			GasPrice:   new(big.Int),
			GasFeeCap:  new(big.Int),
			GasTipCap:  new(big.Int),
			Data:       input,
			AccessList: nil,
			// todo 4844 fields
		}
		_, _ = core.ApplyMessage(evm, msg, &gp)
	})
}
