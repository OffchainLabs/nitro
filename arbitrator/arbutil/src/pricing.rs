// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

/// For hostios that may return something.
pub const HOSTIO_INK: u64 = 8400;

/// For hostios that include pointers.
pub const PTR_INK: u64 = 13440 - HOSTIO_INK;

/// For hostios that involve an API cost.
pub const EVM_API_INK: u64 = 59673;

/// For hostios that involve a div or mod.
pub const DIV_INK: u64 = 20000;

/// For hostios that involve a mulmod.
pub const MUL_MOD_INK: u64 = 24100;

/// For hostios that involve an addmod.
pub const ADD_MOD_INK: u64 = 21000;
