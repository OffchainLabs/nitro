// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import "errors"

// All calls to this precompile are authorized by the DebugPrecompile wrapper,
// which ensures these methods are not accessible in production.
type ArbDebug struct {
	Address      addr                                                     // 0xff
	Basic        func(ctx, mech, bool, bytes32) error                     // index'd: 2nd
	Mixed        func(ctx, mech, bool, bool, bytes32, addr, addr) error   // index'd: 1st 3rd 5th
	Store        func(ctx, mech, bool, addr, huge, bytes32, []byte) error // index'd: 1st 2nd
	BasicGasCost func(bool, bytes32) (uint64, error)
	MixedGasCost func(bool, bool, bytes32, addr, addr) (uint64, error)
	StoreGasCost func(bool, addr, huge, bytes32, []byte) (uint64, error)

	CustomError func(uint64, string, bool) error
	UnusedError func() error
}

func (con ArbDebug) Events(c ctx, evm mech, paid huge, flag bool, value bytes32) (addr, huge, error) {
	// Emits 2 events that cover each case
	//   Basic tests an index'd value & a normal value
	//   Mixed interleaves index'd and normal values that may need to be padded

	err := con.Basic(c, evm, !flag, value)
	if err != nil {
		return addr{}, nil, err
	}

	err = con.Mixed(c, evm, flag, !flag, value, con.Address, c.caller)
	if err != nil {
		return addr{}, nil, err
	}

	return c.caller, paid, nil
}

func (con ArbDebug) CustomRevert(c ctx, number uint64) error {
	return con.CustomError(number, "This spider family wards off bugs: /\\oo/\\ //\\(oo)/\\ /\\oo/\\", true)
}

// Caller becomes a chain owner
func (con ArbDebug) BecomeChainOwner(c ctx, evm mech) error {
	return c.State.ChainOwners().Add(c.caller)
}

func (con ArbDebug) LegacyError(c ctx) error {
	return errors.New("example legacy error")
}
