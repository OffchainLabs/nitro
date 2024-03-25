// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::env::{Escape, HostioInfo, MaybeEscape, WasmEnv, WasmEnvMut};
use arbutil::{
    evm::{
        api::{DataReader, EvmApi},
        EvmData,
    },
    Color,
};
use caller_env::GuestPtr;
use eyre::Result;
use prover::value::Value;
use std::{
    fmt::Display,
    mem::{self, MaybeUninit},
};
use user_host_trait::UserHost;
use wasmer::{MemoryAccessError, WasmPtr};

impl<'a, DR, A> UserHost<DR> for HostioInfo<'a, DR, A>
where
    DR: DataReader,
    A: EvmApi<DR>,
{
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

    fn read_fixed<const N: usize>(
        &self,
        ptr: GuestPtr,
    ) -> std::result::Result<[u8; N], Self::MemoryErr> {
        HostioInfo::read_fixed(self, ptr)
    }

    fn read_slice(&self, ptr: GuestPtr, len: u32) -> Result<Vec<u8>, Self::MemoryErr> {
        let len = len as usize;
        let mut data: Vec<MaybeUninit<u8>> = Vec::with_capacity(len);
        // SAFETY: read_uninit fills all available space
        unsafe {
            data.set_len(len);
            self.view().read_uninit(ptr.into(), &mut data)?;
            Ok(mem::transmute(data))
        }
    }

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) -> Result<(), Self::MemoryErr> {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr.into());
        ptr.deref(&self.view()).write(x)?;
        Ok(())
    }

    fn write_slice(&self, ptr: GuestPtr, src: &[u8]) -> Result<(), Self::MemoryErr> {
        self.view().write(ptr.into(), src)
    }

    fn say<D: Display>(&self, text: D) {
        println!("{} {text}", "Stylus says:".yellow());
    }

    fn trace(&mut self, name: &str, args: &[u8], outs: &[u8], end_ink: u64) {
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

pub(crate) fn read_args<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, read_args(ptr))
}

pub(crate) fn write_result<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
    len: u32,
) -> MaybeEscape {
    hostio!(env, write_result(ptr, len))
}

pub(crate) fn exit_early<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    status: u32,
) -> MaybeEscape {
    hostio!(env, exit_early(status))?;
    Err(Escape::Exit(status))
}

pub(crate) fn storage_load_bytes32<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    key: GuestPtr,
    dest: GuestPtr,
) -> MaybeEscape {
    hostio!(env, storage_load_bytes32(key, dest))
}

pub(crate) fn storage_cache_bytes32<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    key: GuestPtr,
    value: GuestPtr,
) -> MaybeEscape {
    hostio!(env, storage_cache_bytes32(key, value))
}

pub(crate) fn storage_flush_cache<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    clear: u32,
) -> MaybeEscape {
    hostio!(env, storage_flush_cache(clear != 0))
}

pub(crate) fn call_contract<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    value: GuestPtr,
    gas: u64,
    ret_len: GuestPtr,
) -> Result<u8, Escape> {
    hostio!(
        env,
        call_contract(contract, data, data_len, value, gas, ret_len)
    )
}

pub(crate) fn delegate_call_contract<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    gas: u64,
    ret_len: GuestPtr,
) -> Result<u8, Escape> {
    hostio!(
        env,
        delegate_call_contract(contract, data, data_len, gas, ret_len)
    )
}

pub(crate) fn static_call_contract<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    gas: u64,
    ret_len: GuestPtr,
) -> Result<u8, Escape> {
    hostio!(
        env,
        static_call_contract(contract, data, data_len, gas, ret_len)
    )
}

pub(crate) fn create1<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    code: GuestPtr,
    code_len: u32,
    endowment: GuestPtr,
    contract: GuestPtr,
    revert_len: GuestPtr,
) -> MaybeEscape {
    hostio!(
        env,
        create1(code, code_len, endowment, contract, revert_len)
    )
}

pub(crate) fn create2<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    code: GuestPtr,
    code_len: u32,
    endowment: GuestPtr,
    salt: GuestPtr,
    contract: GuestPtr,
    revert_len: GuestPtr,
) -> MaybeEscape {
    hostio!(
        env,
        create2(code, code_len, endowment, salt, contract, revert_len)
    )
}

pub(crate) fn read_return_data<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    dest: GuestPtr,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    hostio!(env, read_return_data(dest, offset, size))
}

pub(crate) fn return_data_size<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u32, Escape> {
    hostio!(env, return_data_size())
}

pub(crate) fn emit_log<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    data: GuestPtr,
    len: u32,
    topics: u32,
) -> MaybeEscape {
    hostio!(env, emit_log(data, len, topics))
}

pub(crate) fn account_balance<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    address: GuestPtr,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, account_balance(address, ptr))
}

pub(crate) fn account_code<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    address: GuestPtr,
    offset: u32,
    size: u32,
    code: GuestPtr,
) -> Result<u32, Escape> {
    hostio!(env, account_code(address, offset, size, code))
}

pub(crate) fn account_code_size<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    address: GuestPtr,
) -> Result<u32, Escape> {
    hostio!(env, account_code_size(address))
}

pub(crate) fn account_codehash<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    address: GuestPtr,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, account_codehash(address, ptr))
}

pub(crate) fn block_basefee<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, block_basefee(ptr))
}

pub(crate) fn block_coinbase<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, block_coinbase(ptr))
}

pub(crate) fn block_gas_limit<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u64, Escape> {
    hostio!(env, block_gas_limit())
}

pub(crate) fn block_number<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u64, Escape> {
    hostio!(env, block_number())
}

pub(crate) fn block_timestamp<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u64, Escape> {
    hostio!(env, block_timestamp())
}

pub(crate) fn chainid<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u64, Escape> {
    hostio!(env, chainid())
}

pub(crate) fn contract_address<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, contract_address(ptr))
}

pub(crate) fn evm_gas_left<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u64, Escape> {
    hostio!(env, evm_gas_left())
}

pub(crate) fn evm_ink_left<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u64, Escape> {
    hostio!(env, evm_ink_left())
}

pub(crate) fn msg_reentrant<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u32, Escape> {
    hostio!(env, msg_reentrant())
}

pub(crate) fn msg_sender<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, msg_sender(ptr))
}

pub(crate) fn msg_value<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, msg_value(ptr))
}

pub(crate) fn native_keccak256<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    input: GuestPtr,
    len: u32,
    output: GuestPtr,
) -> MaybeEscape {
    hostio!(env, native_keccak256(input, len, output))
}

pub(crate) fn tx_gas_price<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, tx_gas_price(ptr))
}

pub(crate) fn tx_ink_price<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
) -> Result<u32, Escape> {
    hostio!(env, tx_ink_price())
}

pub(crate) fn tx_origin<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
) -> MaybeEscape {
    hostio!(env, tx_origin(ptr))
}

pub(crate) fn pay_for_memory_grow<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    pages: u16,
) -> MaybeEscape {
    hostio!(env, pay_for_memory_grow(pages))
}

pub(crate) fn console_log_text<D: DataReader, E: EvmApi<D>>(
    mut env: WasmEnvMut<D, E>,
    ptr: GuestPtr,
    len: u32,
) -> MaybeEscape {
    hostio!(env, console_log_text(ptr, len))
}

pub(crate) fn console_log<D: DataReader, E: EvmApi<D>, T: Into<Value>>(
    mut env: WasmEnvMut<D, E>,
    value: T,
) -> MaybeEscape {
    hostio!(env, console_log(value))
}

pub(crate) fn console_tee<D: DataReader, E: EvmApi<D>, T: Into<Value> + Copy>(
    mut env: WasmEnvMut<D, E>,
    value: T,
) -> Result<T, Escape> {
    hostio!(env, console_tee(value))
}

pub(crate) fn null_host<D: DataReader, E: EvmApi<D>>(_: WasmEnvMut<D, E>) {}
