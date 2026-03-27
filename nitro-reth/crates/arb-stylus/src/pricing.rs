use crate::ink::Ink;

/// Base cost for hostios that may return something.
pub const HOSTIO_INK: Ink = Ink(8400);

/// Extra cost for hostios that include pointers.
pub const PTR_INK: Ink = Ink(13440).sub(HOSTIO_INK);

/// Extra cost for hostios that involve an API cost.
pub const EVM_API_INK: Ink = Ink(59673);

/// Extra cost for division/modulo operations.
pub const DIV_INK: Ink = Ink(20000);

/// Extra cost for mulmod.
pub const MUL_MOD_INK: Ink = Ink(24100);

/// Extra cost for addmod.
pub const ADD_MOD_INK: Ink = Ink(21000);

/// Per-hostio base ink costs.
pub mod hostio {
    use super::*;

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

/// EVM gas constants used by host functions.
pub mod evm_gas {
    /// params.SstoreSentryGasEIP2200
    pub const SSTORE_SENTRY_GAS: u64 = 2300;
    /// params.ColdAccountAccessCostEIP2929
    pub const COLD_ACCOUNT_GAS: u64 = 2600;
    /// params.ColdSloadCostEIP2929
    pub const COLD_SLOAD_GAS: u64 = 2100;
    /// params.WarmStorageReadCostEIP2929
    pub const WARM_SLOAD_GAS: u64 = 100;
    /// params.WarmStorageReadCostEIP2929 (TLOAD cost)
    pub const TLOAD_GAS: u64 = WARM_SLOAD_GAS;
    /// params.WarmStorageReadCostEIP2929 (TSTORE cost)
    pub const TSTORE_GAS: u64 = WARM_SLOAD_GAS;
    /// params.LogGas
    pub const LOG_TOPIC_GAS: u64 = 375;
    /// params.LogDataGas
    pub const LOG_DATA_GAS: u64 = 8;
    /// Minimum gas the cache requires for SSTORE operations.
    pub const STORAGE_CACHE_REQUIRED_ACCESS_GAS: u64 = 100;
}

/// Cost to read `bytes` from WASM memory.
pub fn read_price(bytes: u32) -> Ink {
    Ink(sat_add_mul(16381, 55, bytes.saturating_sub(32)))
}

/// Cost to write `bytes` to WASM memory.
pub fn write_price(bytes: u32) -> Ink {
    Ink(sat_add_mul(5040, 30, bytes.saturating_sub(32)))
}

/// Cost of keccak256 over `bytes`.
pub fn keccak_price(bytes: u32) -> Ink {
    let words = evm_words(bytes).saturating_sub(2);
    Ink(sat_add_mul(121800, 21000, words))
}

/// Cost of exponentiation based on the exponent's byte size.
pub fn pow_price(exponent: &[u8; 32]) -> Ink {
    let mut exp = 33u64;
    for byte in exponent.iter() {
        match *byte == 0 {
            true => exp -= 1,
            false => break,
        }
    }
    Ink(3000 + exp * 17500)
}

fn evm_words(bytes: u32) -> u32 {
    bytes.div_ceil(32)
}

fn sat_add_mul(base: u64, per: u64, count: u32) -> u64 {
    base.saturating_add(per.saturating_mul(count as u64))
}
