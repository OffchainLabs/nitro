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
    sp: GoStack,
) {
    println!("{}", "compile".blue());
    const WASM_PTR: usize = 0;
    const WASM_LEN: usize = 1;
    const _WASM_CAP: usize = 2;
    const CONFIG: usize = 3;
    const STATUS: usize = 4;
    const MACHINE: usize = 5;
    const ERROR: usize = 6;

    macro_rules! error {
        ($msg:expr, $error:expr) => {{
            let error = format!("{}: {:?}", $msg, $error).as_bytes().to_vec();
            sp.write_u32(ERROR, heapify(error));
            sp.write_u32(STATUS, 1);
            return;
        }};
    }

    let wasm = read_go_slice(sp, WASM_PTR, WASM_LEN);
    let config = Box::from_raw(sp.read_u32(CONFIG) as *mut StylusConfig);

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
        Ok(machine) => sp.write_u32(MACHINE, heapify(machine)),
        Err(err) => error!("failed to instrument user program", err),
    }
    sp.write_u32(STATUS, 0);
}

/// Links and executes a user wasm.
///
/// Safety: The go side has the following signature, which must be respected.
/// λ(machine *Machine, calldata []byte, params *StylusConfig, gas *u64) (status userStatus, out *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_callUserWasmRustImpl(
    sp: GoStack,
) {
    println!("{}", "call".blue());
    todo!("callUserWasmRustImpl")
}

/// Reads a rust `Vec`
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(vec *Vec<u8>) (ptr *byte, len usize)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_readRustVecImpl(
    sp: GoStack,
) {
    println!("{}", "read vec".blue());
    let vec = &*(sp.read_u32(0) as *const Vec<u8>);
    sp.write_u32(1, vec.as_ptr() as u32);
    sp.write_u32(2, vec.len() as u32);
}

/// Frees a rust `Vec`.
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(vec *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_freeRustVecImpl(
    sp: GoStack,
) {
    println!("{}", "free vec".blue());
    let vec = Box::from_raw(sp.read_u32(0) as *mut Vec<u8>);
    mem::drop(vec)
}

/// Creates a `StylusConfig` from its component parts.
///
/// SAFETY: The go side has the following signature, which must be respected.
/// λ(version, maxDepth, heapBound u32, wasmGasPrice, hostioCost u64) *StylusConfig
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustConfigImpl(
    sp: GoStack,
) {
    println!("{}", "config".blue());
    let version = sp.read_u32(0);

    let mut config = Box::new(StylusConfig::version(version));
    config.max_depth = sp.read_u32(1);
    config.heap_bound = sp.read_u32(2).into();
    config.pricing.wasm_gas_price = sp.read_u64(3);
    config.pricing.hostio_cost = sp.read_u64(4);

    let handle = Box::into_raw(config) as u32;
    sp.write_u32(5, handle);
}

unsafe fn read_go_slice(sp: GoStack, ptr: usize, len: usize) -> Vec<u8> {
    let wasm_ptr = sp.read_u64(ptr);
    let wasm_len = sp.read_u64(len);
    wavm::read_slice(wasm_ptr, wasm_len)
}

/// Puts an arbitrary type on the heap. The type must be later freed or the value will be leaked.
/// Note: we have a guarantee that wasm won't allocate memory larger than a u32
unsafe fn heapify<T>(value: T) -> u32 {
    Box::into_raw(Box::new(value)) as u32
}
