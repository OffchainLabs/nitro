// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{Bytes20, Bytes32};
use api::Gas;

pub mod api;
pub mod req;
pub mod storage;
pub mod user;

// params.SstoreSentryGasEIP2200
pub const SSTORE_SENTRY_GAS: Gas = Gas(2300);

// params.ColdAccountAccessCostEIP2929
pub const COLD_ACCOUNT_GAS: Gas = Gas(2600);

// params.ColdSloadCostEIP2929
pub const COLD_SLOAD_GAS: Gas = Gas(2100);

// params.WarmStorageReadCostEIP2929
pub const WARM_SLOAD_GAS: Gas = Gas(100);

// params.WarmStorageReadCostEIP2929 (see enable1153 in jump_table.go)
pub const TLOAD_GAS: Gas = WARM_SLOAD_GAS;
pub const TSTORE_GAS: Gas = WARM_SLOAD_GAS;

// params.LogGas and params.LogDataGas
pub const LOG_TOPIC_GAS: Gas = Gas(375);
pub const LOG_DATA_GAS: Gas = Gas(8);

// params.CopyGas
pub const COPY_WORD_GAS: Gas = Gas(3);

// params.Keccak256Gas
pub const KECCAK_256_GAS: Gas = Gas(30);
pub const KECCAK_WORD_GAS: Gas = Gas(6);

// vm.GasQuickStep (see gas.go)
pub const GAS_QUICK_STEP: Gas = Gas(2);

// vm.GasQuickStep (see jump_table.go)
pub const ADDRESS_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see eips.go)
pub const BASEFEE_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see eips.go)
pub const CHAINID_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const COINBASE_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const GASLIMIT_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const NUMBER_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const TIMESTAMP_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const GASLEFT_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const CALLER_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const CALLVALUE_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const GASPRICE_GAS: Gas = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const ORIGIN_GAS: Gas = GAS_QUICK_STEP;

pub const ARBOS_VERSION_STYLUS_CHARGING_FIXES: u64 = 32;
pub const ARBOS_VERSION_STYLUS_LAST_CODE_CACHE_FIX: u64 = 40;

#[derive(Clone, Copy, Debug, Default)]
#[repr(C)]
pub struct EvmData {
    pub arbos_version: u64,
    pub block_basefee: Bytes32,
    pub chainid: u64,
    pub block_coinbase: Bytes20,
    pub block_gas_limit: u64,
    pub block_number: u64,
    pub block_timestamp: u64,
    pub contract_address: Bytes20,
    pub module_hash: Bytes32,
    pub msg_sender: Bytes20,
    pub msg_value: Bytes32,
    pub tx_gas_price: Bytes32,
    pub tx_origin: Bytes20,
    pub reentrant: u32,
    pub return_data_len: u32,
    pub cached: bool,
    pub tracing: bool,
}

/// Returns the minimum number of EVM words needed to store `bytes` bytes.
pub fn evm_words(bytes: u32) -> u32 {
    crate::math::div_ceil::<32>(bytes as usize) as u32
}
