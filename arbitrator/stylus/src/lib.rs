// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::evm_api::GoEvmApi;
use arbutil::{
    evm::{
        user::{UserOutcome, UserOutcomeKind},
        EvmData,
    },
    format::DebugBytes,
};
use eyre::ErrReport;
use native::NativeInstance;
use prover::{programs::prelude::*, Machine};
use run::RunProgram;
use std::mem;

pub use prover;

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

    unsafe fn write_err(&mut self, err: ErrReport) -> UserOutcomeKind {
        self.write(err.debug_bytes());
        UserOutcomeKind::Failure
    }

    unsafe fn write_outcome(&mut self, outcome: UserOutcome) -> UserOutcomeKind {
        let (status, outs) = outcome.into_data();
        self.write(outs);
        status
    }
}

/// Ensures a user program can be proven.
/// On success, `wasm_info` is populated with pricing information.
/// On error, a message is written to `output`.
///
/// # Safety
///
/// `output` and `wasm_info` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_parse_wasm(
    wasm: GoSliceData,
    page_limit: u16,
    version: u16,
    debug: bool,
    wasm_info: *mut WasmPricingInfo,
    output: *mut RustVec,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let info = &mut *wasm_info;
    let output = &mut *output;

    match Machine::new_user_stub(wasm, page_limit, version, debug) {
        Ok((_, data)) => *info = data,
        Err(error) => return output.write_err(error),
    }
    UserOutcomeKind::Success
}

/// Compiles a user program to its native representation.
/// The `output` is either the serialized module or an error string.
///
/// # Safety
///
/// Output must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_compile(
    wasm: GoSliceData,
    version: u16,
    debug_mode: bool,
    output: *mut RustVec,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let output = &mut *output;
    let compile = CompileConfig::version(version, debug_mode);

    let module = match native::module(wasm, compile) {
        Ok(module) => module,
        Err(err) => return output.write_err(err),
    };
    output.write(module);
    UserOutcomeKind::Success
}

/// Calls a compiled user program.
///
/// # Safety
///
/// `module` must represent a valid module produced from `stylus_compile`.
/// `output` and `gas` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_call(
    module: GoSliceData,
    calldata: GoSliceData,
    config: StylusConfig,
    go_api: GoEvmApi,
    evm_data: EvmData,
    debug_chain: u32,
    output: *mut RustVec,
    gas: *mut u64,
) -> UserOutcomeKind {
    let module = module.slice();
    let calldata = calldata.slice().to_vec();
    let compile = CompileConfig::version(config.version, debug_chain != 0);
    let pricing = config.pricing;
    let output = &mut *output;
    let ink = pricing.gas_to_ink(*gas);

    // Safety: module came from compile_user_wasm and we've paid for memory expansion
    let instance = unsafe { NativeInstance::deserialize(module, compile, go_api, evm_data) };
    let mut instance = match instance {
        Ok(instance) => instance,
        Err(error) => panic!("failed to instantiate program: {error:?}"),
    };

    let status = match instance.run_main(&calldata, config, ink) {
        Err(e) | Ok(UserOutcome::Failure(e)) => output.write_err(e.wrap_err("call failed")),
        Ok(outcome) => output.write_outcome(outcome),
    };
    let ink_left = match status {
        UserOutcomeKind::OutOfStack => 0, // take all gas when out of stack
        _ => instance.ink_left().into(),
    };
    *gas = pricing.ink_to_gas(ink_left);
    status
}

/// Frees the vector. Does nothing when the vector is null.
///
/// # Safety
///
/// Must only be called once per vec.
#[no_mangle]
pub unsafe extern "C" fn stylus_drop_vec(vec: RustVec) {
    if !vec.ptr.is_null() {
        mem::drop(vec.into_vec())
    }
}

/// Overwrites the bytes of the vector.
///
/// # Safety
///
/// `rust` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_vec_set_bytes(rust: *mut RustVec, data: GoSliceData) {
    let rust = &mut *rust;
    let mut vec = Vec::from_raw_parts(rust.ptr, rust.len, rust.cap);
    vec.clear();
    vec.extend(data.slice());
    rust.write(vec);
}
