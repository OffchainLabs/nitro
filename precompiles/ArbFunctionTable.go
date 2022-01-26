//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"math/big"
)

// This precompile provided aggregator's the ability to manage function tables.
// Aggregation works differently in Nitro, so these methods have been stubbed and their effects disabled.
// They are kept for backwards compatibility.
type ArbFunctionTable struct {
	Address addr // 0x68
}

// Does nothing
func (con ArbFunctionTable) Upload(c ctx, evm mech, buf []byte) error {
	return nil
}

// Returns the empty table's size, which is 0
func (con ArbFunctionTable) Size(c ctx, evm mech, addr addr) (huge, error) {
	return big.NewInt(0), nil
}

// Reverts since the table is empty
func (con ArbFunctionTable) Get(c ctx, evm mech, addr addr, index huge) (huge, bool, huge, error) {
	return nil, false, nil, errors.New("table is empty")
}
