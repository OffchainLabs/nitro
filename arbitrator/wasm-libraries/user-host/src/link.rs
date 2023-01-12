// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::wavm;
use fnv::FnvHashMap as HashMap;
use go_abi::GoStack;
use prover::{programs::config::StylusConfig, Machine};
use std::{mem, path::Path, sync::Arc};

/// Compiles and instruments user wasm.
/// Safety: λ(wasm []byte, params *StylusConfig) (machine *Machine, err *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_compileUserWasmRustImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let wasm = sp.read_go_slice_owned();
    let config: Box<StylusConfig> = Box::from_raw(sp.read_ptr_mut());

    macro_rules! error {
        ($msg:expr, $error:expr) => {{
            let error = format!("{}: {:?}", $msg, $error).as_bytes().to_vec();
            sp.write_nullptr();
            sp.write_ptr(heapify(error));
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
    let machine = match machine {
        Ok(machine) => machine,
        Err(err) => error!("failed to instrument user program", err),
    };
    sp.write_ptr(heapify(machine));
    sp.write_nullptr();
}

/// Links and executes a user wasm.
/// Safety: λ(machine *Machine, calldata []byte, params *StylusConfig, gas *u64) (status userStatus, out *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_callUserWasmRustImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let machine: Box<Machine> = Box::from_raw(sp.read_ptr_mut());
    let calldata = sp.read_go_slice_owned();
    let config: Box<StylusConfig> = Box::from_raw(sp.read_ptr_mut());
    let gas: *mut u64 = sp.read_ptr_mut();
    
    todo!("callUserWasmRustImpl")
}

/// Reads the length of a rust `Vec`
/// Safety: λ(vec *Vec<u8>) (len u32)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_readRustVecLenImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let vec: &Vec<u8> = &*sp.read_ptr();
    sp.write_u32(vec.len() as u32);
}

/// Copies the contents of a rust `Vec` into a go slice, dropping it in the process
/// Safety: λ(vec *Vec<u8>, dest []byte)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustVecIntoSliceImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let vec: Box<Vec<u8>> = Box::from_raw(sp.read_ptr_mut());
    let ptr: *mut u8 = sp.read_ptr_mut();
    wavm::write_slice(&vec, ptr as u64);
    mem::drop(vec)
}

/// Creates a `StylusConfig` from its component parts.
/// Safety: λ(version, maxDepth, heapBound u32, wasmGasPrice, hostioCost u64) *StylusConfig
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustConfigImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let version = sp.read_u32();

    let mut config = StylusConfig::version(version);
    config.max_depth = sp.read_u32();
    config.heap_bound = sp.read_u32().into();
    config.pricing.wasm_gas_price = sp.skip_space().read_u64();
    config.pricing.hostio_cost = sp.read_u64();
    sp.write_ptr(heapify(config));
}

/// Puts an arbitrary type on the heap. The type must be later freed or the value will be leaked.
unsafe fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}
