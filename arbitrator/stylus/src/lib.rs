// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::evm_api::GoEvmApi;
use arbutil::evm::{
    user::{UserOutcome, UserOutcomeKind},
    EvmData,
};
use eyre::{eyre, ErrReport};
use native::NativeInstance;
use prover::{binary::WasmBinary, programs::prelude::*};
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

    unsafe fn write_err(&mut self, err: ErrReport) {
        self.write(format!("{err:?}").into_bytes());
    }
}

/// Compiles a user program to its native representation.
/// The `output` is either the serialized module or an error string.
///
/// # Safety
///
/// Output must not be null
#[no_mangle]
pub unsafe extern "C" fn stylus_compile(
    wasm: GoSliceData,
    version: u32,
    page_limit: u16,
    footprint: *mut u16,
    output: *mut RustVec,
    debug_mode: usize,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let output = &mut *output;
    let compile = CompileConfig::version(version, debug_mode != 0);

    macro_rules! error {
        ($text:expr, $error:expr) => {
            error!($error.wrap_err($text))
        };
        ($error:expr) => {{
            output.write_err($error);
            return UserOutcomeKind::Failure;
        }};
    }

    // ensure the wasm compiles during proving
    *footprint = match WasmBinary::parse_user(&wasm, page_limit, &compile) {
        Ok((.., pages)) => pages,
        Err(error) => error!("failed to parse program", error),
    };
    let module = match native::module(wasm, compile) {
        Ok(module) => module,
        Err(error) => error!(error),
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
    let ink = pricing.gas_to_ink(*gas);
    let output = &mut *output;

    // Safety: module came from compile_user_wasm
    let instance = unsafe { NativeInstance::deserialize(module, compile, go_api, evm_data) };
    let mut instance = match instance {
        Ok(instance) => instance,
        Err(error) => panic!("failed to instantiate program: {error:?}"),
    };

    let status = match instance.run_main(&calldata, config, ink) {
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

/// Frees the vector.
///
/// # Safety
///
/// Must only be called once per vec.
#[no_mangle]
pub unsafe extern "C" fn stylus_drop_vec(vec: RustVec) {
    mem::drop(vec.into_vec())
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
