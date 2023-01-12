// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{wavm, Color};
use fnv::FnvHashMap as HashMap;
use go_abi::GoStack;
use prover::{programs::prelude::StylusConfig, Machine};
use std::{mem, path::Path, sync::Arc};

/// Compiles and instruments user wasm.
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(wasm []byte, params *StylusConfig) (status userStatus, machine *Machine, err *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_compileUserWasmRustImpl(
    sp: usize,
) {
    println!("{}", "compile".blue());
    let mut sp = GoStack::new(sp);
    let wasm = sp.read_go_slice_owned();
    let config = Box::from_raw(sp.read_u64() as *mut StylusConfig);

    macro_rules! error {
        ($msg:expr, $error:expr) => {{
            let error = format!("{}: {:?}", $msg, $error).as_bytes().to_vec();
            sp.write_u32(1);
            sp.write_u32(heapify(error));
            return;
        }};
    }

    let mut bin = match prover::binary::parse(&wasm, Path::new("user")) {
        Ok(bin) => bin,
        Err(err) => error!("failed to parse user program", err),
    };
    let stylus_data = match bin.instrument(&config) {
        Ok(stylus_data) => stylus_data,
        Err(err) => error!("failed to instrument user program", err),
    };

    let forward = include_bytes!("../../../../target/machines/latest/forward_stub.wasm");
    let forward = prover::binary::parse(forward, Path::new("forward")).unwrap();

    let machine = Machine::from_binaries(
        &[forward],
        bin,
        false,
        false,
        false,
        prover::machine::GlobalState::default(),
        HashMap::default(),
        Arc::new(|_, _| panic!("user program tried to read preimage")),
        Some(stylus_data),
    );
    match machine {
        Ok(machine) => sp.write_u32(heapify(machine)),
        Err(err) => error!("failed to instrument user program", err),
    }
    sp.write_u32(0);
}

/// Links and executes a user wasm.
///
/// Safety: The go side has the following signature, which must be respected.
/// λ(machine *Machine, calldata []byte, params *StylusConfig, gas *u64) (status userStatus, out *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_callUserWasmRustImpl(
    sp: usize,
) {
    //let mut sp = GoStack::new(sp);
    println!("{}", "call".blue());
    todo!("callUserWasmRustImpl")
}

/// Reads a rust `Vec`
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(vec *Vec<u8>) (ptr *byte, len usize)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_readRustVecImpl(
    sp: usize,
) {
    println!("{}", "read vec".blue());
    let mut sp = GoStack::new(sp);
    let vec = &*(sp.read_u32() as *const Vec<u8>);
    sp.write_u32(vec.as_ptr() as u32);
    sp.write_u32(vec.len() as u32);
}

/// Frees a rust `Vec`.
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(vec *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_freeRustVecImpl(
    sp: usize,
) {
    println!("{}", "free vec".blue());
    let mut sp = GoStack::new(sp);
    let vec = Box::from_raw(sp.read_u32() as *mut Vec<u8>);
    mem::drop(vec)
}

/// Creates a `StylusConfig` from its component parts.
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(version, maxDepth, heapBound u32, wasmGasPrice, hostioCost u64) *StylusConfig
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustConfigImpl(
    sp: usize,
) {
    println!("{}", "config".blue());
    let mut sp = GoStack::new(sp);
    let version = sp.read_u32();

    let mut config = Box::new(StylusConfig::version(version));
    config.max_depth = sp.read_u32();
    config.heap_bound = sp.read_u32().into();
    config.pricing.wasm_gas_price = sp.skip_u32().read_u64();
    config.pricing.hostio_cost = sp.read_u64();

    let handle = Box::into_raw(config) as u32;
    sp.write_u32(handle);
}

/// Puts an arbitrary type on the heap. The type must be later freed or the value will be leaked.
/// Note: we have a guarantee that wasm won't allocate memory larger than a u32
unsafe fn heapify<T>(value: T) -> u32 {
    Box::into_raw(Box::new(value)) as u32
}
