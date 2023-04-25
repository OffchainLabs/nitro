// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::evm_api::GoApi;
use eyre::{eyre, ErrReport};
use native::NativeInstance;
use prover::programs::{
    config::{EvmData, GoParams},
    prelude::*,
};
use run::RunProgram;
use std::mem;

pub use {
    crate::evm_api::{EvmApi, EvmApiMethod, EvmApiStatus},
    prover,
};

mod env;
mod evm_api;
pub mod host;
pub mod native;
pub mod run;

#[cfg(test)]
mod test;

#[cfg(all(test, feature = "benchmark"))]
mod benchmarks;

#[repr(C)]
pub struct GoSliceData {
    ptr: *const u8,
    len: usize,
}

impl GoSliceData {
    unsafe fn slice(&self) -> &[u8] {
        std::slice::from_raw_parts(self.ptr, self.len)
    }
}

#[repr(C)]
pub struct RustVec {
    ptr: *mut u8,
    len: usize,
    cap: usize,
}

impl RustVec {
    fn new(vec: Vec<u8>) -> Self {
        let mut rust_vec = Self {
            ptr: std::ptr::null_mut(),
            len: 0,
            cap: 0,
        };
        unsafe { rust_vec.write(vec) };
        rust_vec
    }

    unsafe fn into_vec(self) -> Vec<u8> {
        Vec::from_raw_parts(self.ptr, self.len, self.cap)
    }

    unsafe fn write(&mut self, mut vec: Vec<u8>) {
        self.ptr = vec.as_mut_ptr();
        self.len = vec.len();
        self.cap = vec.capacity();
        mem::forget(vec);
    }

    unsafe fn write_err(&mut self, err: ErrReport) {
        self.write(format!("{err:?}").into_bytes());
    }
}

#[no_mangle]
pub unsafe extern "C" fn stylus_compile(
    wasm: GoSliceData,
    version: u32,
    debug_mode: usize,
    output: *mut RustVec,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let output = &mut *output;
    let config = CompileConfig::version(version, debug_mode != 0);

    match native::module(wasm, config) {
        Ok(module) => {
            output.write(module);
            UserOutcomeKind::Success
        }
        Err(error) => {
            output.write_err(error);
            UserOutcomeKind::Failure
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn stylus_call(
    module: GoSliceData,
    calldata: GoSliceData,
    params: GoParams,
    go_api: GoApi,
    evm_data: EvmData,
    output: *mut RustVec,
    gas: *mut u64,
) -> UserOutcomeKind {
    let module = module.slice();
    let calldata = calldata.slice().to_vec();
    let (compile_config, stylus_config) = params.configs();
    let pricing = stylus_config.pricing;
    let ink = pricing.gas_to_ink(*gas);
    let output = &mut *output;

    // Safety: module came from compile_user_wasm
    let instance = unsafe { NativeInstance::deserialize(module, compile_config, go_api, evm_data) };
    let mut instance = match instance {
        Ok(instance) => instance,
        Err(error) => panic!("failed to instantiate program: {error:?}"),
    };

    let status = match instance.run_main(&calldata, stylus_config, ink) {
        Err(err) | Ok(UserOutcome::Failure(err)) => {
            output.write_err(err.wrap_err(eyre!("failed to execute program")));
            UserOutcomeKind::Failure
        }
        Ok(outcome) => {
            let (status, outs) = outcome.into_data();
            output.write(outs);
            status
        }
    };
    let ink_left = match status {
        UserOutcomeKind::OutOfStack => 0, // take all gas when out of stack
        _ => instance.ink_left().into(),
    };
    *gas = pricing.ink_to_gas(ink_left);
    status
}

#[no_mangle]
pub unsafe extern "C" fn stylus_drop_vec(vec: RustVec) {
    mem::drop(vec.into_vec())
}

#[no_mangle]
pub unsafe extern "C" fn stylus_vec_set_bytes(rust: *mut RustVec, data: GoSliceData) {
    let rust = &mut *rust;
    let mut vec = Vec::from_raw_parts(rust.ptr, rust.len, rust.cap);
    vec.clear();
    vec.extend(data.slice());
    rust.write(vec);
}
