// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    abi::Bytes,
    alloy_primitives::{Address, B256, U256},
    block, console, contract, evm, hostio, msg,
    prelude::*,
    stylus_proc::entrypoint,
    tx,
    types::AddressVM,
};
extern crate alloc;

#[cfg(target_arch = "wasm32")]
#[global_allocator]
static ALLOC: mini_alloc::MiniAlloc = mini_alloc::MiniAlloc::INIT;

sol_storage! {
    #[entrypoint]
    pub struct HostioTest {
    }
}

type Result<T> = std::result::Result<T, Vec<u8>>;

// These are not available as hostios in the sdk, so we import them directly.
#[link(wasm_import_module = "vm_hooks")]
extern "C" {
    fn math_div(value: *mut u8, divisor: *const u8);
    fn math_mod(value: *mut u8, modulus: *const u8);
    fn math_pow(value: *mut u8, exponent: *const u8);
    fn math_add_mod(value: *mut u8, addend: *const u8, modulus: *const u8);
    fn math_mul_mod(value: *mut u8, multiplier: *const u8, modulus: *const u8);
    fn transient_load_bytes32(key: *const u8, dest: *mut u8);
    fn transient_store_bytes32(key: *const u8, value: *const u8);
    fn exit_early(status: u32);
}

#[external]
impl HostioTest {
    fn exit_early() -> Result<()> {
        unsafe {
            exit_early(0);
        }
        Ok(())
    }

    fn transient_load_bytes32(key: B256) -> Result<B256> {
        let mut result = B256::ZERO;
        unsafe {
            transient_load_bytes32(key.as_ptr(), result.as_mut_ptr());
        }
        Ok(result)
    }

    fn transient_store_bytes32(key: B256, value: B256) {
        unsafe {
            transient_store_bytes32(key.as_ptr(), value.as_ptr());
        }
    }

    fn return_data_size() -> Result<U256> {
        unsafe { Ok(hostio::return_data_size().try_into().unwrap()) }
    }

    fn emit_log(data: Bytes, n: i8, t1: B256, t2: B256, t3: B256, t4: B256) -> Result<()> {
        let topics = &[t1, t2, t3, t4];
        evm::raw_log(&topics[0..n as usize], data.as_slice())?;
        Ok(())
    }

    fn account_balance(account: Address) -> Result<U256> {
        Ok(account.balance())
    }

    fn account_code(account: Address) -> Result<Vec<u8>> {
        let mut size = 10000;
        let mut code = vec![0; size];
        unsafe {
            size = hostio::account_code(account.as_ptr(), 0, size, code.as_mut_ptr());
        }
        code.resize(size, 0);
        Ok(code)
    }

    fn account_code_size(account: Address) -> Result<U256> {
        Ok(account.code_size().try_into().unwrap())
    }

    fn account_codehash(account: Address) -> Result<B256> {
        Ok(account.codehash())
    }

    fn evm_gas_left() -> Result<U256> {
        Ok(evm::gas_left().try_into().unwrap())
    }

    fn evm_ink_left() -> Result<U256> {
        Ok(tx::ink_to_gas(evm::ink_left()).try_into().unwrap())
    }

    fn block_basefee() -> Result<U256> {
        Ok(block::basefee())
    }

    fn chainid() -> Result<U256> {
        Ok(block::chainid().try_into().unwrap())
    }

    fn block_coinbase() -> Result<Address> {
        Ok(block::coinbase())
    }

    fn block_gas_limit() -> Result<U256> {
        Ok(block::gas_limit().try_into().unwrap())
    }

    fn block_number() -> Result<U256> {
        Ok(block::number().try_into().unwrap())
    }

    fn block_timestamp() -> Result<U256> {
        Ok(block::timestamp().try_into().unwrap())
    }

    fn contract_address() -> Result<Address> {
        Ok(contract::address())
    }

    fn math_div(a: U256, b: U256) -> Result<U256> {
        let mut a_bytes: B256 = a.into();
        let b_bytes: B256 = b.into();
        unsafe {
            math_div(a_bytes.as_mut_ptr(), b_bytes.as_ptr());
        }
        Ok(a_bytes.into())
    }

    fn math_mod(a: U256, b: U256) -> Result<U256> {
        let mut a_bytes: B256 = a.into();
        let b_bytes: B256 = b.into();
        unsafe {
            math_mod(a_bytes.as_mut_ptr(), b_bytes.as_ptr());
        }
        Ok(a_bytes.into())
    }

    fn math_pow(a: U256, b: U256) -> Result<U256> {
        let mut a_bytes: B256 = a.into();
        let b_bytes: B256 = b.into();
        unsafe {
            math_pow(a_bytes.as_mut_ptr(), b_bytes.as_ptr());
        }
        Ok(a_bytes.into())
    }

    fn math_add_mod(a: U256, b: U256, c: U256) -> Result<U256> {
        let mut a_bytes: B256 = a.into();
        let b_bytes: B256 = b.into();
        let c_bytes: B256 = c.into();
        unsafe {
            math_add_mod(a_bytes.as_mut_ptr(), b_bytes.as_ptr(), c_bytes.as_ptr());
        }
        Ok(a_bytes.into())
    }

    fn math_mul_mod(a: U256, b: U256, c: U256) -> Result<U256> {
        let mut a_bytes: B256 = a.into();
        let b_bytes: B256 = b.into();
        let c_bytes: B256 = c.into();
        unsafe {
            math_mul_mod(a_bytes.as_mut_ptr(), b_bytes.as_ptr(), c_bytes.as_ptr());
        }
        Ok(a_bytes.into())
    }

    fn msg_sender() -> Result<Address> {
        Ok(msg::sender())
    }

    fn msg_value() -> Result<U256> {
        Ok(msg::value())
    }

    fn keccak(preimage: Bytes) -> Result<B256> {
        let mut result = B256::ZERO;
        unsafe {
            hostio::native_keccak256(preimage.as_ptr(), preimage.len(), result.as_mut_ptr());
        }
        Ok(result)
    }

    fn tx_gas_price() -> Result<U256> {
        Ok(tx::gas_price())
    }

    fn tx_ink_price() -> Result<U256> {
        Ok(tx::ink_to_gas(tx::ink_price().into()).try_into().unwrap())
    }

    fn tx_origin() -> Result<Address> {
        Ok(tx::origin())
    }

    fn storage_cache_bytes32() {
        let key = B256::ZERO;
        let val = B256::ZERO;
        unsafe {
            hostio::storage_cache_bytes32(key.as_ptr(), val.as_ptr());
        }
    }

    fn pay_for_memory_grow(pages: U256) {
        let pages: u16 = pages.try_into().unwrap();
        unsafe {
            hostio::pay_for_memory_grow(pages);
        }
    }

    fn write_result_empty() {
    }

    fn write_result(size: U256) -> Result<Vec<u32>> {
        let size: usize = size.try_into().unwrap();
        let data = vec![0; size];
        Ok(data)
    }

    fn read_args_no_args() {
    }

    fn read_args_one_arg(_arg1: U256) {
    }

    fn read_args_three_args(_arg1: U256, _arg2: U256, _arg3: U256) {
    }
}
