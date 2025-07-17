// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{
    evm::{self, api::Ink},
    Bytes32,
};

/// For hostios that may return something.
pub const HOSTIO_INK: Ink = Ink(8400);

/// For hostios that include pointers.
pub const PTR_INK: Ink = Ink(13440).sub(HOSTIO_INK);

/// For hostios that involve an API cost.
pub const EVM_API_INK: Ink = Ink(59673);

/// For hostios that involve a div or mod.
pub const DIV_INK: Ink = Ink(20000);

/// For hostios that involve a mulmod.
pub const MUL_MOD_INK: Ink = Ink(24100);

/// For hostios that involve an addmod.
pub const ADD_MOD_INK: Ink = Ink(21000);

/// Defines the price of each Hostio.
pub mod hostio {
    pub use super::*;

    pub const READ_ARGS_BASE_INK: Ink = HOSTIO_INK;
    pub const WRITE_RESULT_BASE_INK: Ink = HOSTIO_INK;
    pub const STORAGE_LOAD_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(2));
    pub const STORAGE_CACHE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(2));
    pub const STORAGE_FLUSH_BASE_INK: Ink = HOSTIO_INK.add(EVM_API_INK);
    pub const TRANSIENT_LOAD_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(2).add(EVM_API_INK));
    pub const TRANSIENT_STORE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(2).add(EVM_API_INK));
    pub const CALL_CONTRACT_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(3).add(EVM_API_INK));
    pub const CREATE1_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(3).add(EVM_API_INK));
    pub const CREATE2_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(4).add(EVM_API_INK));
    pub const READ_RETURN_DATA_BASE_INK: Ink = HOSTIO_INK.add(EVM_API_INK);
    pub const RETURN_DATA_SIZE_BASE_INK: Ink = HOSTIO_INK;
    pub const EMIT_LOG_BASE_INK: Ink = HOSTIO_INK.add(EVM_API_INK);
    pub const ACCOUNT_BALANCE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(2).add(EVM_API_INK));
    pub const ACCOUNT_CODE_BASE_INK: Ink = HOSTIO_INK.add(EVM_API_INK);
    pub const ACCOUNT_CODE_SIZE_BASE_INK: Ink = HOSTIO_INK.add(EVM_API_INK);
    pub const ACCOUNT_CODE_HASH_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(2).add(EVM_API_INK));
    pub const BLOCK_BASEFEE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const BLOCK_COINBASE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const BLOCK_GAS_LIMIT_BASE_INK: Ink = HOSTIO_INK;
    pub const BLOCK_NUMBER_BASE_INK: Ink = HOSTIO_INK;
    pub const BLOCK_TIMESTAMP_BASE_INK: Ink = HOSTIO_INK;
    pub const CHAIN_ID_BASE_INK: Ink = HOSTIO_INK;
    pub const ADDRESS_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const EVM_GAS_LEFT_BASE_INK: Ink = HOSTIO_INK;
    pub const EVM_INK_LEFT_BASE_INK: Ink = HOSTIO_INK;
    pub const MATH_DIV_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(3).add(DIV_INK));
    pub const MATH_MOD_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(3).add(DIV_INK));
    pub const MATH_POW_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(3));
    pub const MATH_ADD_MOD_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(4).add(ADD_MOD_INK));
    pub const MATH_MUL_MOD_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK.mul(4).add(MUL_MOD_INK));
    pub const MSG_REENTRANT_BASE_INK: Ink = HOSTIO_INK;
    pub const MSG_SENDER_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const MSG_VALUE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const TX_GAS_PRICE_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const TX_INK_PRICE_BASE_INK: Ink = HOSTIO_INK;
    pub const TX_ORIGIN_BASE_INK: Ink = HOSTIO_INK.add(PTR_INK);
    pub const PAY_FOR_MEMORY_GROW_BASE_INK: Ink = HOSTIO_INK;
}

pub fn write_price(bytes: u32) -> Ink {
    Ink(sat_add_mul(5040, 30, bytes.saturating_sub(32)))
}

pub fn read_price(bytes: u32) -> Ink {
    Ink(sat_add_mul(16381, 55, bytes.saturating_sub(32)))
}

pub fn keccak_price(bytes: u32) -> Ink {
    let words = evm::evm_words(bytes).saturating_sub(2);
    Ink(sat_add_mul(121800, 21000, words))
}

pub fn pow_price(exponent: &Bytes32) -> Ink {
    let mut exp = 33;
    for byte in exponent.iter() {
        match *byte == 0 {
            true => exp -= 1, // reduce cost for each big-endian 0 byte
            false => break,
        }
    }
    Ink(3000 + exp * 17500)
}

fn sat_add_mul(base: u64, per: u64, count: u32) -> u64 {
    base.saturating_add(per.saturating_mul(count.into()))
}
