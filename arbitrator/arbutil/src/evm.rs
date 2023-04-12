// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

// params.SstoreSentryGasEIP2200
pub const SSTORE_SENTRY_EVM_GAS: u64 = 2300;

// params.LogGas and params.LogDataGas
pub const LOG_TOPIC_GAS: u64 = 375;
pub const LOG_DATA_GAS: u64 = 8;

// params.CopyGas
pub const COPY_WORD_GAS: u64 = 3;

// vm.GasQuickStep (see eips.go)
pub const BASEFEE_GAS: u64 = 2;

// vm.GasQuickStep (see eips.go)
pub const CHAINID_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const COINBASE_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const DIFFICULTY_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const GASLIMIT_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const NUMBER_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const TIMESTAMP_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const GASLEFT_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const CALLER_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const CALLVALUE_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const GASPRICE_GAS: u64 = 2;

// vm.GasQuickStep (see jump_table.go)
pub const ORIGIN_GAS: u64 = 2;
