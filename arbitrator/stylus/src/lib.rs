// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::{
    evm::{
        api::{DataReader, Gas, Ink},
        req::EvmApiRequestor,
        user::{UserOutcome, UserOutcomeKind},
        EvmData,
    },
    format::DebugBytes,
    Bytes32,
};
use cache::{deserialize_module, CacheMetrics, InitCache};
use evm_api::NativeRequestHandler;
use eyre::ErrReport;
use native::NativeInstance;
use prover::{
    programs::{prelude::*, StylusData},
    RustBytes,
};
use run::RunProgram;
use std::ptr;
use target_cache::{target_cache_get, target_cache_set};

pub use brotli;
pub use prover;

pub mod env;
pub mod host;
pub mod native;
pub mod run;

mod cache;
mod evm_api;
mod target_cache;
mod util;

#[cfg(test)]
mod test;

#[cfg(all(test, feature = "benchmark"))]
mod benchmarks;

#[derive(Clone, Copy)]
#[repr(C)]
pub struct GoSliceData {
    /// Points to data owned by Go.
    ptr: *const u8,
    /// The length in bytes.
    len: usize,
}

/// The data we're pointing to is owned by Go and has a lifetime no shorter than the current program.
unsafe impl Send for GoSliceData {}

impl GoSliceData {
    pub fn null() -> Self {
        Self {
            ptr: ptr::null(),
            len: 0,
        }
    }

    fn slice(&self) -> &[u8] {
        if self.len == 0 {
            return &[];
        }
        unsafe { std::slice::from_raw_parts(self.ptr, self.len) }
    }
}

impl DataReader for GoSliceData {
    fn slice(&self) -> &[u8] {
        if self.len == 0 {
            return &[];
        }
        unsafe { std::slice::from_raw_parts(self.ptr, self.len) }
    }
}

unsafe fn write_err(output: &mut RustBytes, err: ErrReport) -> UserOutcomeKind {
    output.write(err.debug_bytes());
    UserOutcomeKind::Failure
}

unsafe fn write_outcome(output: &mut RustBytes, outcome: UserOutcome) -> UserOutcomeKind {
    let (status, outs) = outcome.into_data();
    output.write(outs);
    status
}

/// "activates" a user wasm.
///
/// The `output` is either the module or an error string.
/// Returns consensus info such as the module hash and footprint on success.
///
/// Note that this operation costs gas and is limited by the amount supplied via the `gas` pointer.
/// The amount left is written back at the end of the call.
///
/// # Safety
///
/// `output`, `asm_len`, `module_hash`, `footprint`, and `gas` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_activate(
    wasm: GoSliceData,
    page_limit: u16,
    stylus_version: u16,
    arbos_version_for_gas: u64,
    debug: bool,
    output: *mut RustBytes,
    codehash: *const Bytes32,
    module_hash: *mut Bytes32,
    stylus_data: *mut StylusData,
    gas: *mut u64,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let output = &mut *output;
    let module_hash = &mut *module_hash;
    let codehash = &*codehash;
    let gas = &mut *gas;

    let (module, info) = match native::activate(
        wasm,
        codehash,
        stylus_version,
        arbos_version_for_gas,
        page_limit,
        debug,
        gas,
    ) {
        Ok(val) => val,
        Err(err) => return write_err(output, err),
    };

    *module_hash = module.hash();
    *stylus_data = info;

    output.write(module.into_bytes());
    UserOutcomeKind::Success
}

/// "compiles" a user wasm.
///
/// The `output` is either the asm or an error string.
/// Returns consensus info such as the module hash and footprint on success.
///
/// # Safety
///
/// `output` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_compile(
    wasm: GoSliceData,
    version: u16,
    debug: bool,
    name: GoSliceData,
    output: *mut RustBytes,
) -> UserOutcomeKind {
    let wasm = wasm.slice();
    let output = &mut *output;
    let name = match String::from_utf8(name.slice().to_vec()) {
        Ok(val) => val,
        Err(err) => return write_err(output, err.into()),
    };
    let target = match target_cache_get(&name) {
        Ok(val) => val,
        Err(err) => return write_err(output, err),
    };

    let asm = match native::compile(wasm, version, debug, target) {
        Ok(val) => val,
        Err(err) => return write_err(output, err),
    };

    output.write(asm);
    UserOutcomeKind::Success
}

#[no_mangle]
/// # Safety
///
/// `output` must not be null.
pub unsafe extern "C" fn wat_to_wasm(wat: GoSliceData, output: *mut RustBytes) -> UserOutcomeKind {
    let output = &mut *output;
    let wasm = match wasmer::wat2wasm(wat.slice()) {
        Ok(val) => val,
        Err(err) => return write_err(output, err.into()),
    };
    output.write(wasm.into_owned());
    UserOutcomeKind::Success
}

/// sets target index to a string
///
/// String format is: Triple+CpuFeature+CpuFeature..
///
/// # Safety
///
/// `output` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_target_set(
    name: GoSliceData,
    description: GoSliceData,
    output: *mut RustBytes,
    native: bool,
) -> UserOutcomeKind {
    let output = &mut *output;
    let name = match String::from_utf8(name.slice().to_vec()) {
        Ok(val) => val,
        Err(err) => return write_err(output, err.into()),
    };

    let desc_str = match String::from_utf8(description.slice().to_vec()) {
        Ok(val) => val,
        Err(err) => return write_err(output, err.into()),
    };

    if let Err(err) = target_cache_set(name, desc_str, native) {
        return write_err(output, err);
    };

    UserOutcomeKind::Success
}

/// Calls an activated user program.
///
/// # Safety
///
/// `module` must represent a valid module produced from `stylus_activate`.
/// `output` and `gas` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_call(
    module: GoSliceData,
    calldata: GoSliceData,
    config: StylusConfig,
    req_handler: NativeRequestHandler,
    evm_data: EvmData,
    debug_chain: bool,
    output: *mut RustBytes,
    gas: *mut u64,
    long_term_tag: u32,
) -> UserOutcomeKind {
    let module = module.slice();
    let calldata = calldata.slice().to_vec();
    let evm_api = EvmApiRequestor::new(req_handler);
    let pricing = config.pricing;
    let output = &mut *output;
    let ink = pricing.gas_to_ink(Gas(*gas));

    // Safety: module came from compile_user_wasm and we've paid for memory expansion
    let instance = unsafe {
        NativeInstance::deserialize_cached(
            module,
            config.version,
            evm_api,
            evm_data,
            long_term_tag,
            debug_chain,
        )
    };
    let mut instance = match instance {
        Ok(instance) => instance,
        Err(error) => util::panic_with_wasm(module, error.wrap_err("init failed")),
    };

    let status = match instance.run_main(&calldata, config, ink) {
        Err(e) | Ok(UserOutcome::Failure(e)) => write_err(output, e.wrap_err("call failed")),
        Ok(outcome) => write_outcome(output, outcome),
    };
    let ink_left = match status {
        UserOutcomeKind::OutOfStack => Ink(0), // take all gas when out of stack
        _ => instance.ink_left().into(),
    };
    *gas = pricing.ink_to_gas(ink_left).0;
    status
}

/// set lru cache capacity
#[no_mangle]
pub extern "C" fn stylus_set_cache_lru_capacity(capacity_bytes: u64) {
    InitCache::set_lru_capacity(capacity_bytes);
}

/// Caches an activated user program.
///
/// # Safety
///
/// `module` must represent a valid module produced from `stylus_activate`.
/// arbos_tag: a tag for arbos cache. 0 won't affect real caching
/// currently only if tag==1 caching will be affected
#[no_mangle]
pub unsafe extern "C" fn stylus_cache_module(
    module: GoSliceData,
    module_hash: Bytes32,
    version: u16,
    arbos_tag: u32,
    debug: bool,
) {
    if let Err(error) = InitCache::insert(module_hash, module.slice(), version, arbos_tag, debug) {
        panic!("tried to cache invalid asm!: {error}");
    }
}

/// Evicts an activated user program from the init cache.
#[no_mangle]
pub extern "C" fn stylus_evict_module(
    module_hash: Bytes32,
    version: u16,
    arbos_tag: u32,
    debug: bool,
) {
    InitCache::evict(module_hash, version, arbos_tag, debug);
}

/// Reorgs the init cache. This will likely never happen.
#[no_mangle]
pub extern "C" fn stylus_reorg_vm(_block: u64, arbos_tag: u32) {
    InitCache::clear_long_term(arbos_tag);
}

/// Gets cache metrics.
///
/// # Safety
///
/// `output` must not be null.
#[no_mangle]
pub unsafe extern "C" fn stylus_get_cache_metrics(output: *mut CacheMetrics) {
    let output = &mut *output;
    InitCache::get_metrics(output);
}

/// Clears lru cache.
/// Only used for testing purposes.
#[no_mangle]
pub extern "C" fn stylus_clear_lru_cache() {
    InitCache::clear_lru_cache()
}

/// Clears long term cache (for arbos_tag = 1)
/// Only used for testing purposes.
#[no_mangle]
pub extern "C" fn stylus_clear_long_term_cache() {
    InitCache::clear_long_term(1);
}

/// Gets entry size in bytes.
/// Only used for testing purposes.
#[no_mangle]
pub extern "C" fn stylus_get_entry_size_estimate_bytes(
    module: GoSliceData,
    version: u16,
    debug: bool,
) -> u64 {
    match deserialize_module(module.slice(), version, debug) {
        Err(error) => panic!("tried to get invalid asm!: {error}"),
        Ok((_, _, entry_size_estimate_bytes)) => entry_size_estimate_bytes.try_into().unwrap(),
    }
}
