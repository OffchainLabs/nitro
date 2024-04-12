// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

pub mod api;
pub mod req;
pub mod storage;
pub mod user;

// params.SstoreSentryGasEIP2200
pub const SSTORE_SENTRY_GAS: u64 = 2300;

// params.ColdAccountAccessCostEIP2929
pub const COLD_ACCOUNT_GAS: u64 = 2600;

// params.ColdSloadCostEIP2929
pub const COLD_SLOAD_GAS: u64 = 2100;

// params.WarmStorageReadCostEIP2929
pub const WARM_SLOAD_GAS: u64 = 100;

// params.WarmStorageReadCostEIP2929 (see enable1153 in jump_table.go)
pub const TLOAD_GAS: u64 = WARM_SLOAD_GAS;
pub const TSTORE_GAS: u64 = WARM_SLOAD_GAS;

// params.LogGas and params.LogDataGas
pub const LOG_TOPIC_GAS: u64 = 375;
pub const LOG_DATA_GAS: u64 = 8;

// params.CopyGas
pub const COPY_WORD_GAS: u64 = 3;

// params.Keccak256Gas
pub const KECCAK_256_GAS: u64 = 30;
pub const KECCAK_WORD_GAS: u64 = 6;

// vm.GasQuickStep (see gas.go)
pub const GAS_QUICK_STEP: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const ADDRESS_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see eips.go)
pub const BASEFEE_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see eips.go)
pub const CHAINID_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const COINBASE_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const GASLIMIT_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const NUMBER_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const TIMESTAMP_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const GASLEFT_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const CALLER_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const CALLVALUE_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const GASPRICE_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const ORIGIN_GAS: u64 = GAS_QUICK_STEP;

#[derive(Clone, Copy, Debug, Default)]
#[repr(C)]
pub struct EvmData {
    pub block_basefee: Bytes32,
    pub chainid: u64,
    pub block_coinbase: Bytes20,
    pub block_gas_limit: u64,
    pub block_number: u64,
    pub block_timestamp: u64,
    pub contract_address: Bytes20,
    pub msg_sender: Bytes20,
    pub msg_value: Bytes32,
    pub tx_gas_price: Bytes32,
    pub tx_origin: Bytes20,
    pub reentrant: u32,
    pub return_data_len: u32,
    pub tracing: bool,
}

/// Returns the minimum number of EVM words needed to store `bytes` bytes.
pub fn evm_words(bytes: u32) -> u32 {
    match bytes % 32 {
        0 => bytes / 32,
        _ => bytes / 32 + 1,
    }
}
