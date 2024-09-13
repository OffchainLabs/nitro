use std::fmt::Display;

pub enum ParamType {
    I32,
    I64,
}

impl Display for ParamType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        use ParamType::*;
        match self {
            I32 => write!(f, "i32"),
            I64 => write!(f, "i64"),
        }
    }
}

use ParamType::*;

/// order matters!
pub const HOSTIOS: [(&str, &[ParamType], &[ParamType]); 42] = [
    ("read_args", &[I32], &[]),
    ("write_result", &[I32, I32], &[]),
    ("exit_early", &[I32], &[]),
    ("storage_load_bytes32", &[I32, I32], &[]),
    ("storage_cache_bytes32", &[I32, I32], &[]),
    ("storage_flush_cache", &[I32], &[]),
    ("transient_load_bytes32", &[I32, I32], &[]),
    ("transient_store_bytes32", &[I32, I32], &[]),
    ("call_contract", &[I32, I32, I32, I32, I64, I32], &[I32]),
    ("delegate_call_contract", &[I32, I32, I32, I64, I32], &[I32]),
    ("static_call_contract", &[I32, I32, I32, I64, I32], &[I32]),
    ("create1", &[I32, I32, I32, I32, I32], &[]),
    ("create2", &[I32, I32, I32, I32, I32, I32], &[]),
    ("read_return_data", &[I32, I32, I32], &[I32]),
    ("return_data_size", &[], &[I32]),
    ("emit_log", &[I32, I32, I32], &[]),
    ("account_balance", &[I32, I32], &[]),
    ("account_code", &[I32, I32, I32, I32], &[I32]),
    ("account_code_size", &[I32], &[I32]),
    ("account_codehash", &[I32, I32], &[]),
    ("evm_gas_left", &[], &[I64]),
    ("evm_ink_left", &[], &[I64]),
    ("block_basefee", &[I32], &[]),
    ("chainid", &[], &[I64]),
    ("block_coinbase", &[I32], &[]),
    ("block_gas_limit", &[], &[I64]),
    ("block_number", &[], &[I64]),
    ("block_timestamp", &[], &[I64]),
    ("contract_address", &[I32], &[]),
    ("math_div", &[I32, I32], &[]),
    ("math_mod", &[I32, I32], &[]),
    ("math_pow", &[I32, I32], &[]),
    ("math_add_mod", &[I32, I32, I32], &[]),
    ("math_mul_mod", &[I32, I32, I32], &[]),
    ("msg_reentrant", &[], &[I32]),
    ("msg_sender", &[I32], &[]),
    ("msg_value", &[I32], &[]),
    ("native_keccak256", &[I32, I32, I32], &[]),
    ("tx_gas_price", &[I32], &[]),
    ("tx_ink_price", &[], &[I32]),
    ("tx_origin", &[I32], &[]),
    ("pay_for_memory_grow", &[I32], &[]),
];
