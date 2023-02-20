// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::{eyre, ErrReport};
use native::NativeInstance;
use prover::{programs::prelude::*, utils::Bytes32};
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
}

impl GoParams {
    pub fn config(self) -> StylusConfig {
        let mut config = StylusConfig::version(self.version);
        config.depth.max_depth = self.max_depth;
        config.pricing.wasm_gas_price = self.wasm_gas_price;
        config.pricing.hostio_cost = self.hostio_cost;
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
    ptr: *mut *mut u8,
    len: *mut usize,
    cap: *mut usize,
}

impl RustVec {
    unsafe fn write(&mut self, mut vec: Vec<u8>) {
        let ptr = vec.as_mut_ptr();
        let len = vec.len();
        let cap = vec.capacity();
        mem::forget(vec);
        *self.ptr = ptr;
        *self.len = len;
        *self.cap = cap;
    }

    unsafe fn write_err(&mut self, err: ErrReport) {
        let msg = format!("{:?}", err);
        let vec = msg.into_bytes();
        self.write(vec)
    }
}

#[no_mangle]
pub unsafe extern "C" fn stylus_compile(
    wasm: GoSliceData,
    version: u32,
    mut output: RustVec,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let config = StylusConfig::version(version);

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
    pub set_bytes32: unsafe extern "C" fn(usize, Bytes32, Bytes32, *mut u64) -> usize,
    pub id: usize,
}

#[no_mangle]
pub unsafe extern "C" fn stylus_call(
    module: GoSliceData,
    calldata: GoSliceData,
    params: GoParams,
    go_api: GoAPI,
    mut output: RustVec,
    evm_gas: *mut u64,
) -> UserOutcomeKind {
    let module = module.slice();
    let calldata = calldata.slice().to_vec();
    let config = params.config();
    let pricing = config.pricing;
    let wasm_gas = pricing.evm_to_wasm(*evm_gas).unwrap_or(u64::MAX);

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
    let vec = Vec::from_raw_parts(*vec.ptr, *vec.len, *vec.cap);
    mem::drop(vec)
}
