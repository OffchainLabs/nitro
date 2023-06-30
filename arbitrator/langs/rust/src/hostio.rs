// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn account_balance(address: *const u8, dest: *mut u8);
    pub(crate) fn account_codehash(address: *const u8, dest: *mut u8);
    pub(crate) fn block_basefee(basefee: *mut u8);
    pub(crate) fn block_chainid(chainid: *mut u8);
    pub(crate) fn block_coinbase(coinbase: *mut u8);
    pub(crate) fn block_difficulty(difficulty: *mut u8);
    pub(crate) fn block_gas_limit() -> u64;
    pub(crate) fn block_number(number: *mut u8);
    pub(crate) fn block_timestamp() -> u64;
    pub(crate) fn call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        value: *const u8,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;
    pub(crate) fn create1(
        code: *const u8,
        code_len: usize,
        endowment: *const u8,
        contract: *mut u8,
        revert_data_len: *mut usize,
    );
    pub(crate) fn create2(
        code: *const u8,
        code_len: usize,
        endowment: *const u8,
        salt: *const u8,
        contract: *mut u8,
        revert_data_len: *mut usize,
    );
    pub(crate) fn delegate_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;
    pub(crate) fn emit_log(data: *const u8, len: usize, topics: usize);
    pub(crate) fn evm_blockhash(number: *const u8, dest: *mut u8);
    pub(crate) fn evm_gas_left() -> u64;
    pub(crate) fn evm_ink_left() -> u64;
    pub(crate) fn msg_sender(sender: *mut u8);
    pub(crate) fn msg_value(value: *mut u8);
    pub(crate) fn read_args(dest: *mut u8);
    /// A noop when there's never been a call
    pub(crate) fn read_return_data(dest: *mut u8);
    pub(crate) fn return_data(data: *const u8, len: usize);
    /// Returns 0 when there's never been a call
    pub(crate) fn return_data_size() -> u32;
    pub(crate) fn static_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;
    pub(crate) fn tx_gas_price(gas_price: *mut u8);
    pub(crate) fn tx_ink_price() -> u64;
    pub(crate) fn tx_origin(origin: *mut u8);
}

#[allow(dead_code)]
#[link(wasm_import_module = "console")]
extern "C" {
    pub(crate) fn log_f32(value: f32);
    pub(crate) fn log_f64(value: f64);
    pub(crate) fn log_i32(value: i32);
    pub(crate) fn log_i64(value: i64);
    pub(crate) fn log_txt(text: *const u8, len: usize);
}
