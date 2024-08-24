// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import "github.com/ethereum/go-ethereum/common"

type ArbWasmCache struct {
	Address addr // 0x72

	UpdateProgramCache        func(ctx, mech, addr, bytes32, bool) error
	UpdateProgramCacheGasCost func(addr, bytes32, bool) (uint64, error)
}

// See if the user is a cache manager owner.
func (con ArbWasmCache) IsCacheManager(c ctx, _ mech, addr addr) (bool, error) {
	return c.State.Programs().CacheManagers().IsMember(addr)
}

// Retrieve all authorized address managers.
func (con ArbWasmCache) AllCacheManagers(c ctx, _ mech) ([]addr, error) {
	return c.State.Programs().CacheManagers().AllMembers(65536)
}

// Deprecated: replaced with CacheProgram.
func (con ArbWasmCache) CacheCodehash(c ctx, evm mech, codehash hash) error {
	return con.setProgramCached(c, evm, common.Address{}, codehash, true)
}

// Caches all programs with a codehash equal to the given address. Caller must be a cache manager or chain owner.
func (con ArbWasmCache) CacheProgram(c ctx, evm mech, address addr) error {
	codehash, err := c.GetCodeHash(address)
	if err != nil {
		return err
	}
	return con.setProgramCached(c, evm, address, codehash, true)
}

// Evicts all programs with the given codehash. Caller must be a cache manager or chain owner.
func (con ArbWasmCache) EvictCodehash(c ctx, evm mech, codehash hash) error {
	return con.setProgramCached(c, evm, common.Address{}, codehash, false)
}

// Gets whether a program is cached. Note that the program may be expired.
func (con ArbWasmCache) CodehashIsCached(c ctx, evm mech, codehash hash) (bool, error) {
	return c.State.Programs().ProgramCached(codehash)
}

// Caches all programs with the given codehash.
func (con ArbWasmCache) setProgramCached(c ctx, evm mech, address addr, codehash hash, cached bool) error {
	if !con.hasAccess(c) {
		return c.BurnOut()
	}
	programs := c.State.Programs()
	params, err := programs.Params()
	if err != nil {
		return err
	}
	debugMode := evm.ChainConfig().DebugMode()
	txRunMode := c.txProcessor.RunMode()
	emitEvent := func() error {
		return con.UpdateProgramCache(c, evm, c.caller, codehash, cached)
	}
	return programs.SetProgramCached(
		emitEvent, evm.StateDB, codehash, address, cached, evm.Context.Time, params, txRunMode, debugMode,
	)
}

func (con ArbWasmCache) hasAccess(c ctx) bool {
	manager, err := c.State.Programs().CacheManagers().IsMember(c.caller)
	if err != nil {
		return false
	}
	if manager {
		return true
	}
	owner, err := c.State.ChainOwners().IsMember(c.caller)
	return owner && err == nil
}
