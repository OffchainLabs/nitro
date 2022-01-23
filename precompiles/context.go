//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"log"
	"math/big"

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/burn"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type addr = common.Address
type mech = *vm.EVM
type huge = *big.Int
type hash = common.Hash
type bytes32 = [32]byte
type ctx = *context

type context struct {
	caller      addr
	gasSupplied uint64
	gasLeft     uint64
	txProcessor *arbos.TxProcessor
	state       *arbosState.ArbosState
	write       bool
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
	return !c.write
}

func testContext(caller addr, evm mech) *context {
	ctx := &context{
		caller:      caller,
		gasSupplied: ^uint64(0),
		gasLeft:     ^uint64(0),
		write:       true,
	}
	state, _ := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(true))
	ctx.state = state
	return ctx
}
