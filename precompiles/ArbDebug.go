//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

type ArbDebug struct {
	Address      addr
	Basic        func(mech, bool, [32]byte)                   // index'd: 2nd
	Mixed        func(mech, bool, bool, [32]byte, addr, addr) // index'd: 1st 3rd 5th
	BasicGasCost func(bool, [32]byte) uint64
	MixedGasCost func(bool, bool, [32]byte, addr, addr) uint64
}

func (con ArbDebug) Events(caller addr, evm mech, paid huge, flag bool, value [32]byte) (addr, huge, error) {
	// Emits 2 events that cover each case
	//   Basic tests an index'd value & a normal value
	//   Mixed interleaves index'd and normal values that may need to be padded

	con.Basic(evm, !flag, value)
	con.Mixed(evm, flag, !flag, value, con.Address, caller)

	return caller, paid, nil
}

func (con ArbDebug) EventsGasCost(flag bool, value [32]byte) uint64 {
	return uint64(value[0])
}
