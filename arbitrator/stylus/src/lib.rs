// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::ErrReport;
use prover::programs::config::StylusConfig;
use wasmer::Bytes;
use std::mem;

mod env;
pub mod stylus;

#[cfg(test)]
mod test;

#[cfg(all(test, feature = "benchmark"))]
mod benchmarks;

#[repr(u8)]
pub enum StylusStatus {
    Success,
    Failure,
}

#[repr(C)]
pub struct GoParams {
    version: u32,
    max_depth: u32,
    heap_bound: u32,
    wasm_gas_price: u64,
    hostio_cost: u64,
}

impl GoParams {
    fn config(self) -> StylusConfig {
        let mut config = StylusConfig::version(self.version);
        config.max_depth = self.max_depth;
        config.heap_bound = Bytes(self.heap_bound as usize);
        config.pricing.wasm_gas_price = self.wasm_gas_price;
        config.pricing.hostio_cost = self.hostio_cost;
        config
    }
}

#[repr(C)]
pub struct GoSlice {
    ptr: *const u8,
    len: usize,
}

impl GoSlice {
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
        let vec = msg.as_bytes().to_vec();
        self.write(vec)
    }
}

#[no_mangle]
pub unsafe extern "C" fn stylus_compile(wasm: GoSlice, params: GoParams, mut output: RustVec) -> StylusStatus {
    let wasm = wasm.slice();
    let config = params.config();

    match stylus::module(wasm, config) {
        Ok(module) => {
            output.write(module);
            StylusStatus::Success
        }
        Err(error) => {
            output.write_err(error);
            StylusStatus::Failure
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn stylus_call(
    module: GoSlice,
    calldata: GoSlice,
    params: GoParams,
    mut output: RustVec,
    gas: *mut u64,
) -> StylusStatus {
    let module = module.slice();
    let calldata = calldata.slice();
    let config = params.config();
    let gas_left = *gas;

    *gas = gas_left;
    output.write_err(ErrReport::msg("not ready"));
    StylusStatus::Failure
}

#[no_mangle]
pub unsafe extern "C" fn stylus_free(vec: RustVec) {
    let vec = Vec::from_raw_parts(*vec.ptr, *vec.len, *vec.cap);
    mem::drop(vec)
}
