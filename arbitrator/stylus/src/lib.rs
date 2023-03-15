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
    wasm_gas_price: u64,
    hostio_cost: u64,
    debug_mode: usize,
}

impl GoParams {
    pub fn config(self) -> StylusConfig {
        let mut config = StylusConfig::version(self.version);
        config.depth.max_depth = self.max_depth;
        config.pricing.wasm_gas_price = self.wasm_gas_price;
        config.pricing.hostio_cost = self.hostio_cost;
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

#[repr(C)]
pub struct GoAPI {
    pub get_bytes32: unsafe extern "C" fn(usize, Bytes32, *mut u64) -> Bytes32,
    pub set_bytes32: unsafe extern "C" fn(usize, Bytes32, Bytes32, *mut u64) -> u8,
    pub call_contract:
        unsafe extern "C" fn(usize, Bytes20, *mut RustVec, *mut u64, Bytes32) -> UserOutcomeKind,
    pub id: usize,
}

#[no_mangle]
pub unsafe extern "C" fn stylus_call(
    module: GoSliceData,
    calldata: GoSliceData,
    params: GoParams,
    go_api: GoAPI,
    output: *mut RustVec,
    evm_gas: *mut u64,
) -> UserOutcomeKind {
    let module = module.slice();
    let calldata = calldata.slice().to_vec();
    let config = params.config();
    let pricing = config.pricing;
    let wasm_gas = pricing.evm_to_wasm(*evm_gas).unwrap_or(u64::MAX);
    let output = &mut *output;

    macro_rules! error {
        ($msg:expr, $report:expr) => {{
            let report: ErrReport = $report.into();
            let report = report.wrap_err(eyre!($msg));
            output.write_err(report);
            *evm_gas = 0; // burn all gas
            return UserOutcomeKind::Failure;
        }};
    }

    // Safety: module came from compile_user_wasm
    let instance = unsafe { NativeInstance::deserialize(module, config.clone()) };

    let mut instance = match instance {
        Ok(instance) => instance,
        Err(error) => error!("failed to instantiate program", error),
    };
    instance.set_go_api(go_api);
    instance.set_gas(wasm_gas);

    let (status, outs) = match instance.run_main(&calldata, &config) {
        Err(err) | Ok(UserOutcome::Failure(err)) => error!("failed to execute program", err),
        Ok(outcome) => outcome.into_data(),
    };
    if pricing.wasm_gas_price != 0 {
        let wasm_gas = match status {
            UserOutcomeKind::OutOfStack => 0, // take all gas when out of stack
            _ => instance.gas_left().into(),
        };
        *evm_gas = pricing.wasm_to_evm(wasm_gas);
    }
    output.write(outs);
    status
}

#[no_mangle]
pub unsafe extern "C" fn stylus_free(vec: RustVec) {
    mem::drop(vec.into_vec())
}

#[no_mangle]
pub unsafe extern "C" fn stylus_overwrite_vec(rust: *mut RustVec, data: GoSliceData) {
    let rust = &mut *rust;
    let mut vec = Vec::from_raw_parts(rust.ptr, rust.len, rust.cap);
    vec.clear();
    vec.extend(data.slice());
    rust.write(vec);
}
