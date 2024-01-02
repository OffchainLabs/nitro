// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use std::fmt::Display;

use crate::env::{Escape, HostioInfo, MaybeEscape, WasmEnv, WasmEnvMut};
use arbutil::{
    evm::{api::EvmApi, EvmData},
    Bytes20, Bytes32, Color,
};
use eyre::Result;
use prover::value::Value;
use user_host_trait::UserHost;
use wasmer::{MemoryAccessError, WasmPtr};

impl<'a, A: EvmApi> UserHost for HostioInfo<'a, A> {
    type Err = Escape;
    type MemoryErr = MemoryAccessError;
    type A = A;

    fn args(&self) -> &[u8] {
        &self.args
    }

    fn outs(&mut self) -> &mut Vec<u8> {
        &mut self.outs
    }

    fn evm_api(&mut self) -> &mut Self::A {
        &mut self.evm_api
    }

    fn evm_data(&self) -> &EvmData {
        &self.evm_data
    }

    fn evm_return_data_len(&mut self) -> &mut u32 {
        &mut self.evm_data.return_data_len
    }

    fn read_bytes20(&self, ptr: u32) -> Result<Bytes20, Self::MemoryErr> {
        let data = self.read_fixed(ptr)?;
        Ok(data.into())
    }

    fn read_bytes32(&self, ptr: u32) -> Result<Bytes32, Self::MemoryErr> {
        let data = self.read_fixed(ptr)?;
        Ok(data.into())
    }

    fn read_slice(&self, ptr: u32, len: u32) -> Result<Vec<u8>, Self::MemoryErr> {
        let mut data = vec![0; len as usize];
        self.view().read(ptr.into(), &mut data)?;
        Ok(data)
    }

    fn write_u32(&mut self, ptr: u32, x: u32) -> Result<(), Self::MemoryErr> {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x)?;
        Ok(())
    }

    fn write_bytes20(&self, ptr: u32, src: Bytes20) -> Result<(), Self::MemoryErr> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }

    fn write_bytes32(&self, ptr: u32, src: Bytes32) -> Result<(), Self::MemoryErr> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }

    fn write_slice(&self, ptr: u32, src: &[u8]) -> Result<(), Self::MemoryErr> {
        self.view().write(ptr.into(), src)
    }

    fn say<D: Display>(&self, text: D) {
        println!("{} {text}", "Stylus says:".yellow());
    }

    fn trace(&self, name: &str, args: &[u8], outs: &[u8], end_ink: u64) {
        let start_ink = self.start_ink;
        self.evm_api
            .capture_hostio(name, args, outs, start_ink, end_ink);
    }
}

macro_rules! hostio {
    ($env:expr, $($func:tt)*) => {
        WasmEnv::program(&mut $env)?.$($func)*
    };
}

pub(crate) fn read_args<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, read_args(ptr))
}

pub(crate) fn write_result<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32, len: u32) -> MaybeEscape {
    hostio!(env, write_result(ptr, len))
}

pub(crate) fn storage_load_bytes32<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    key: u32,
    dest: u32,
) -> MaybeEscape {
    hostio!(env, storage_load_bytes32(key, dest))
}

pub(crate) fn storage_store_bytes32<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    key: u32,
    value: u32,
) -> MaybeEscape {
    hostio!(env, storage_store_bytes32(key, value))
}

pub(crate) fn call_contract<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    value: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    hostio!(
        env,
        call_contract(contract, data, data_len, value, gas, ret_len)
    )
}

pub(crate) fn delegate_call_contract<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    hostio!(
        env,
        delegate_call_contract(contract, data, data_len, gas, ret_len)
    )
}

pub(crate) fn static_call_contract<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    hostio!(
        env,
        static_call_contract(contract, data, data_len, gas, ret_len)
    )
}

pub(crate) fn create1<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    contract: u32,
    revert_len: u32,
) -> MaybeEscape {
    hostio!(
        env,
        create1(code, code_len, endowment, contract, revert_len)
    )
}

pub(crate) fn create2<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    salt: u32,
    contract: u32,
    revert_len: u32,
) -> MaybeEscape {
    hostio!(
        env,
        create2(code, code_len, endowment, salt, contract, revert_len)
    )
}

pub(crate) fn read_return_data<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    dest: u32,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    hostio!(env, read_return_data(dest, offset, size))
}

pub(crate) fn return_data_size<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    hostio!(env, return_data_size())
}

pub(crate) fn emit_log<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    data: u32,
    len: u32,
    topics: u32,
) -> MaybeEscape {
    hostio!(env, emit_log(data, len, topics))
}

pub(crate) fn account_balance<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    ptr: u32,
) -> MaybeEscape {
    hostio!(env, account_balance(address, ptr))
}

pub(crate) fn account_code<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    offset: u32,
    size: u32,
    code: u32,
) -> Result<u32, Escape> {
    hostio!(env, account_code(address, offset, size, code))
}

pub(crate) fn account_codehash<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    ptr: u32,
) -> MaybeEscape {
    hostio!(env, account_codehash(address, ptr))
}

pub(crate) fn account_code_size<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
) -> Result<u32, Escape> {
    hostio!(env, account_code_size(address))
}

pub(crate) fn block_basefee<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, block_basefee(ptr))
}

pub(crate) fn block_coinbase<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, block_coinbase(ptr))
}

pub(crate) fn block_gas_limit<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    hostio!(env, block_gas_limit())
}

pub(crate) fn block_number<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    hostio!(env, block_number())
}

pub(crate) fn block_timestamp<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    hostio!(env, block_timestamp())
}

pub(crate) fn chainid<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    hostio!(env, chainid())
}

pub(crate) fn contract_address<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, contract_address(ptr))
}

pub(crate) fn evm_gas_left<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    hostio!(env, evm_gas_left())
}

pub(crate) fn evm_ink_left<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    hostio!(env, evm_ink_left())
}

pub(crate) fn msg_reentrant<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    hostio!(env, msg_reentrant())
}

pub(crate) fn msg_sender<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, msg_sender(ptr))
}

pub(crate) fn msg_value<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, msg_value(ptr))
}

pub(crate) fn native_keccak256<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    input: u32,
    len: u32,
    output: u32,
) -> MaybeEscape {
    hostio!(env, native_keccak256(input, len, output))
}

pub(crate) fn tx_gas_price<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, tx_gas_price(ptr))
}

pub(crate) fn tx_ink_price<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    hostio!(env, tx_ink_price())
}

pub(crate) fn tx_origin<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    hostio!(env, tx_origin(ptr))
}

pub(crate) fn pay_for_memory_grow<E: EvmApi>(mut env: WasmEnvMut<E>, pages: u16) -> MaybeEscape {
    hostio!(env, pay_for_memory_grow(pages))
}

pub(crate) fn console_log_text<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    ptr: u32,
    len: u32,
) -> MaybeEscape {
    hostio!(env, console_log_text(ptr, len))
}

pub(crate) fn console_log<E: EvmApi, T: Into<Value>>(
    mut env: WasmEnvMut<E>,
    value: T,
) -> MaybeEscape {
    hostio!(env, console_log(value))
}

pub(crate) fn console_tee<E: EvmApi, T: Into<Value> + Copy>(
    mut env: WasmEnvMut<E>,
    value: T,
) -> Result<T, Escape> {
    hostio!(env, console_tee(value))
}

pub(crate) fn null_host<E: EvmApi>(_: WasmEnvMut<E>) {}
