// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::{eyre, ErrReport};
use native::NativeInstance;
use prover::{
    programs::prelude::*,
    utils::{Bytes20, Bytes32},
};
use run::RunProgram;
use std::mem;

use crate::env::EvmData;
pub use prover;

mod env;
pub mod host;
pub mod native;
pub mod run;

#[cfg(test)]
mod test;

#[cfg(all(test, feature = "benchmark"))]
mod benchmarks;

#[repr(C)]
pub struct GoParams {
    version: u32,
    max_depth: u32,
    ink_price: u64,
    hostio_cost: u64,
    debug_mode: usize,
}

impl GoParams {
    pub fn config(self) -> StylusConfig {
        let mut config = StylusConfig::version(self.version);
        config.depth.max_depth = self.max_depth;
        config.pricing.ink_price = self.ink_price;
        config.pricing.hostio_ink = self.hostio_cost;
        config.debug.debug_funcs = self.debug_mode != 0;
        config
    }
}

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

    let mut config = StylusConfig::version(version);
    if debug_mode != 0 {
        config.debug.debug_funcs = true;
    }

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

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum GoApiStatus {
    Success,
    Failure,
}

impl From<GoApiStatus> for UserOutcomeKind {
    fn from(value: GoApiStatus) -> Self {
        match value {
            GoApiStatus::Success => UserOutcomeKind::Success,
            GoApiStatus::Failure => UserOutcomeKind::Revert,
        }
    }
}

#[repr(C)]
pub struct GoApi {
    pub block_hash: unsafe extern "C" fn(id: usize, block: Bytes32, evm_cost: *mut u64) -> Bytes32, // value
    pub get_bytes32: unsafe extern "C" fn(id: usize, key: Bytes32, evm_cost: *mut u64) -> Bytes32, // value
    pub set_bytes32: unsafe extern "C" fn(
        id: usize,
        key: Bytes32,
        value: Bytes32,
        evm_cost: *mut u64,
        error: *mut RustVec,
    ) -> GoApiStatus,
    pub contract_call: unsafe extern "C" fn(
        id: usize,
        contract: Bytes20,
        calldata: *mut RustVec,
        gas: *mut u64,
        value: Bytes32,
        return_data_len: *mut u32,
    ) -> GoApiStatus,
    pub delegate_call: unsafe extern "C" fn(
        id: usize,
        contract: Bytes20,
        calldata: *mut RustVec,
        gas: *mut u64,
        return_data_len: *mut u32,
    ) -> GoApiStatus,
    pub static_call: unsafe extern "C" fn(
        id: usize,
        contract: Bytes20,
        calldata: *mut RustVec,
        gas: *mut u64,
        return_data_len: *mut u32,
    ) -> GoApiStatus,
    pub create1: unsafe extern "C" fn(
        id: usize,
        code: *mut RustVec,
        endowment: Bytes32,
        gas: *mut u64,
        return_data_len: *mut u32,
    ) -> GoApiStatus,
    pub create2: unsafe extern "C" fn(
        id: usize,
        code: *mut RustVec,
        endowment: Bytes32,
        salt: Bytes32,
        gas: *mut u64,
        return_data_len: *mut u32,
    ) -> GoApiStatus,
    pub get_return_data: unsafe extern "C" fn(id: usize, output: *mut RustVec),
    pub emit_log: unsafe extern "C" fn(id: usize, data: *mut RustVec, topics: usize) -> GoApiStatus,
    pub id: usize,
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
    let config = params.config();
    let pricing = config.pricing;
    let ink = pricing.gas_to_ink(*gas);
    let output = &mut *output;

    // Safety: module came from compile_user_wasm
    let instance = unsafe { NativeInstance::deserialize(module, config.clone()) };
    let mut instance = match instance {
        Ok(instance) => instance,
        Err(error) => panic!("failed to instantiate program: {error:?}"),
    };
    instance.set_go_api(go_api);
    instance.set_evm_data(evm_data);
    instance.set_ink(ink);

    let status = match instance.run_main(&calldata, &config) {
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
