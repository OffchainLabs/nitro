//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"log"
	"math/big"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type addr = common.Address
type mech = *vm.EVM
type huge = *big.Int
type hash = common.Hash
type bytes4 = [4]byte
type bytes32 = [32]byte
type ctx = *context

type context struct {
	caller      addr
	gasSupplied uint64
	gasLeft     uint64
	txProcessor *arbos.TxProcessor
	state       *arbosState.ArbosState
	readOnly    bool
}

func (c *context) Burn(amount uint64) error {
	if c.gasLeft < amount {
		c.gasLeft = 0
		return vm.ErrOutOfGas
	}
	c.gasLeft -= amount
	return nil
}

//nolint:unused
func (c *context) burned() uint64 {
	return c.gasSupplied - c.gasLeft
}

func (c *context) Restrict(err error) {
	log.Fatal("A metered burner was used for access-controlled work", err)
}

func (c *context) ReadOnly() bool {
	return c.readOnly
}

func testContext(caller addr, evm mech) *context {
	ctx := &context{
		caller:      caller,
		gasSupplied: ^uint64(0),
		gasLeft:     ^uint64(0),
		readOnly:    false,
	}
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(false))
	if err != nil {
		panic(err)
	}
	ctx.state = state
	ctx.txProcessor = evm.ProcessingHook.(*arbos.TxProcessor)
	return ctx
}
