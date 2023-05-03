// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

pub mod api;
pub mod js;
pub mod user;

// vm.GasQuickStep (see gas.go)
const GAS_QUICK_STEP: u64 = 2;

// params.SstoreSentryGasEIP2200
pub const SSTORE_SENTRY_GAS: u64 = 2300;

// params.LogGas and params.LogDataGas
pub const LOG_TOPIC_GAS: u64 = 375;
pub const LOG_DATA_GAS: u64 = 8;

// params.CopyGas
pub const COPY_WORD_GAS: u64 = 3;

// vm.GasQuickStep (see jump_table.go)
pub const ADDRESS_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see eips.go)
pub const BASEFEE_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see eips.go)
pub const CHAINID_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const COINBASE_GAS: u64 = GAS_QUICK_STEP;

// vm.GasQuickStep (see jump_table.go)
pub const DIFFICULTY_GAS: u64 = GAS_QUICK_STEP;

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
    pub block_chainid: Bytes32,
    pub block_coinbase: Bytes20,
    pub block_difficulty: Bytes32,
    pub block_gas_limit: u64,
    pub block_number: Bytes32,
    pub block_timestamp: Bytes32,
    pub contract_address: Bytes20,
    pub msg_sender: Bytes20,
    pub msg_value: Bytes32,
    pub gas_price: Bytes32,
    pub origin: Bytes20,
    pub return_data_len: u32,
}

impl EvmData {
    pub fn new(
        block_basefee: Bytes32,
        block_chainid: Bytes32,
        block_coinbase: Bytes20,
        block_difficulty: Bytes32,
        block_gas_limit: u64,
        block_number: Bytes32,
        block_timestamp: Bytes32,
        contract_address: Bytes20,
        msg_sender: Bytes20,
        msg_value: Bytes32,
        gas_price: Bytes32,
        origin: Bytes20,
    ) -> Self {
        Self {
            block_basefee,
            block_chainid,
            block_coinbase,
            block_difficulty,
            block_gas_limit,
            block_number,
            block_timestamp,
            contract_address,
            msg_sender,
            msg_value,
            gas_price,
            origin,
            return_data_len: 0,
        }
    }
}
