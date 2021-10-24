//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

type ArbDebug struct {
	Address addr
	Basic   func(mech, bool, [32]byte)
	Spill   func(mech, bool, [2][32]byte)
	Mixed   func(mech, bool, bool, [32]byte, addr, addr)
}

func (con ArbDebug) Events(caller addr, evm mech, paid huge, flag bool, value [32]byte) (addr, huge, error) {
	// Emits 3 events that cover each case
	//   Basic tests an index'd value & a normal value
	//   Spill tests that a value wider than 32 bytes gets hashed when indexing
	//   Mixed interleaves index'd and normal values that may need to be padded

	con.Basic(evm, flag, value)
	con.Spill(evm, flag, ([2][32]byte{value, value}))
	con.Mixed(evm, flag, !flag, value, con.Address, caller)

	return caller, paid, nil
}

func (con ArbDebug) EventsGasCost(flag bool, value [32]byte) uint64 {
	return uint64(value[0])
}
