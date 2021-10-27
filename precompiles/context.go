//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
)

type addr = common.Address
type mech = *vm.EVM
type huge = *big.Int
type ctx = *context

type context struct {
	caller      addr
	gasSupplied uint64
	gasLeft     uint64
}

func (c *context) burn(amount uint64) error {
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

func testContext(caller addr) *context {
	return &context{
		caller:      caller,
		gasSupplied: ^uint64(0),
		gasLeft:     ^uint64(0),
	}
}
