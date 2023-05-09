// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

pub mod binary;
mod host;
pub mod machine;
/// cbindgen:ignore
mod memory;
mod merkle;
mod reinterpret;
pub mod utils;
pub mod value;
pub mod wavm;

use crate::machine::{argument_data_to_inbox, Machine};
use eyre::Result;
use machine::{get_empty_preimage_resolver, GlobalState, MachineStatus, PreimageResolver};
use sha3::{Digest, Keccak256};
use static_assertions::const_assert_eq;
use std::{
    ffi::CStr,
    os::raw::{c_char, c_int},
    path::Path,
    sync::{
        atomic::{self, AtomicU8},
        Arc,
    },
};
use utils::{Bytes32, CBytes};

#[repr(C)]
#[derive(Clone, Copy)]
pub struct CByteArray {
    pub ptr: *const u8,
    pub len: usize,
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct RustByteArray {
    pub ptr: *mut u8,
    pub len: usize,
    pub capacity: usize,
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_load_machine(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
) -> *mut Machine {
    match arbitrator_load_machine_impl(binary_path, library_paths, library_paths_size) {
        Ok(mach) => mach,
        Err(err) => {
            eprintln!("Error loading binary: {}", err);
            std::ptr::null_mut()
        }
    }
}

unsafe fn arbitrator_load_machine_impl(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
) -> Result<*mut Machine> {
    let binary_path = cstr_to_string(binary_path);
    let binary_path = Path::new(&binary_path);

    let mut libraries = vec![];
    for i in 0..library_paths_size {
        let path = cstr_to_string(*(library_paths.offset(i)));
        libraries.push(Path::new(&path).to_owned());
    }

    let mach = Machine::from_paths(
        &libraries,
        binary_path,
        true,
        false,
        false,
        Default::default(),
        Default::default(),
        get_empty_preimage_resolver(),
    )?;
    Ok(Box::into_raw(Box::new(mach)))
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_load_wavm_binary(binary_path: *const c_char) -> *mut Machine {
    let binary_path = cstr_to_string(binary_path);
    let binary_path = Path::new(&binary_path);
    match Machine::new_from_wavm(binary_path) {
        Ok(mach) => Box::into_raw(Box::new(mach)),
        Err(err) => {
            eprintln!("Error loading binary: {}", err);
            std::ptr::null_mut()
        }
    }
}

unsafe fn cstr_to_string(c_str: *const c_char) -> String {
    CStr::from_ptr(c_str).to_string_lossy().into_owned()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_free_machine(mach: *mut Machine) {
    drop(Box::from_raw(mach));
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_clone_machine(mach: *mut Machine) -> *mut Machine {
    let new_mach = (*mach).clone();
    Box::into_raw(Box::new(new_mach))
}

/// Go doesn't have this functionality builtin for whatever reason. Uses relaxed ordering.
#[no_mangle]
pub unsafe extern "C" fn atomic_u8_store(ptr: *mut u8, contents: u8) {
    (*(ptr as *mut AtomicU8)).store(contents, atomic::Ordering::Relaxed);
}

fn err_to_c_string(err: eyre::Report) -> *mut libc::c_char {
    let err = format!("{:#}", err);
    unsafe {
        let buf = libc::malloc(err.len() + 1);
        if buf.is_null() {
            panic!("Failed to allocate memory for error string");
        }
        std::ptr::copy_nonoverlapping(err.as_ptr(), buf as *mut u8, err.len());
        *(buf.add(err.len()) as *mut u8) = 0;
        buf as *mut libc::c_char
    }
}

/// Runs the machine while the condition variable is zero. May return early if num_steps is hit.
/// Returns a c string error (freeable with libc's free) on error, or nullptr on success.
#[no_mangle]
pub unsafe extern "C" fn arbitrator_step(
    mach: *mut Machine,
    num_steps: u64,
    condition: *const u8,
) -> *mut libc::c_char {
    let mach = &mut *mach;
    let condition = &*(condition as *const AtomicU8);
    let mut remaining_steps = num_steps;
    while condition.load(atomic::Ordering::Relaxed) == 0 {
        if remaining_steps == 0 || mach.is_halted() {
            break;
        }
        let stepping = std::cmp::min(remaining_steps, 1_000_000);
        match mach.step_n(stepping) {
            Ok(()) => {}
            Err(err) => return err_to_c_string(err),
        }
        remaining_steps -= stepping;
    }
    std::ptr::null_mut()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_add_inbox_message(
    mach: *mut Machine,
    inbox_identifier: u64,
    index: u64,
    data: CByteArray,
) -> c_int {
    let mach = &mut *mach;
    if let Some(identifier) = argument_data_to_inbox(inbox_identifier) {
        let slice = std::slice::from_raw_parts(data.ptr, data.len);
        let data = slice.to_vec();
        mach.add_inbox_msg(identifier, index, data);
        0
    } else {
        1
    }
}

/// Like arbitrator_step, but stops early if it hits a host io operation.
/// Returns a c string error (freeable with libc's free) on error, or nullptr on success.
#[no_mangle]
pub unsafe extern "C" fn arbitrator_step_until_host_io(
    mach: *mut Machine,
    condition: *const u8,
) -> *mut libc::c_char {
    let mach = &mut *mach;
    let condition = &*(condition as *const AtomicU8);
    while condition.load(atomic::Ordering::Relaxed) == 0 {
        for _ in 0..1_000_000 {
            if mach.is_halted() {
                return std::ptr::null_mut();
            }
            if mach.next_instruction_is_host_io() {
                return std::ptr::null_mut();
            }
            match mach.step_n(1) {
                Ok(()) => {}
                Err(err) => return err_to_c_string(err),
            }
        }
    }
    std::ptr::null_mut()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_serialize_state(
    mach: *const Machine,
    path: *const c_char,
) -> c_int {
    let mach = &*mach;
    let res = CStr::from_ptr(path)
        .to_str()
        .map_err(eyre::Report::from)
        .and_then(|path| mach.serialize_state(path));
    if let Err(err) = res {
        eprintln!("Failed to serialize machine state: {}", err);
        1
    } else {
        0
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_deserialize_and_replace_state(
    mach: *mut Machine,
    path: *const c_char,
) -> c_int {
    let mach = &mut *mach;
    let res = CStr::from_ptr(path)
        .to_str()
        .map_err(eyre::Report::from)
        .and_then(|path| mach.deserialize_and_replace_state(path));
    if let Err(err) = res {
        eprintln!("Failed to deserialize machine state: {}", err);
        1
    } else {
        0
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_get_num_steps(mach: *const Machine) -> u64 {
    (*mach).get_steps()
}

pub const ARBITRATOR_MACHINE_STATUS_RUNNING: u8 = 0;
pub const ARBITRATOR_MACHINE_STATUS_FINISHED: u8 = 1;
pub const ARBITRATOR_MACHINE_STATUS_ERRORED: u8 = 2;
pub const ARBITRATOR_MACHINE_STATUS_TOO_FAR: u8 = 3;

// Unfortunately, cbindgen doesn't support constants with non-literal values, so we assert that they're correct.
const_assert_eq!(
    ARBITRATOR_MACHINE_STATUS_RUNNING,
    MachineStatus::Running as u8,
);
const_assert_eq!(
    ARBITRATOR_MACHINE_STATUS_FINISHED,
    MachineStatus::Finished as u8,
);
const_assert_eq!(
    ARBITRATOR_MACHINE_STATUS_ERRORED,
    MachineStatus::Errored as u8,
);
const_assert_eq!(
    ARBITRATOR_MACHINE_STATUS_TOO_FAR,
    MachineStatus::TooFar as u8,
);

/// Returns one of ARBITRATOR_MACHINE_STATUS_*
#[no_mangle]
pub unsafe extern "C" fn arbitrator_get_status(mach: *const Machine) -> u8 {
    (*mach).get_status() as u8
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_global_state(mach: *mut Machine) -> GlobalState {
    (*mach).get_global_state()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_set_global_state(mach: *mut Machine, gs: GlobalState) {
    (*mach).set_global_state(gs);
}

#[repr(C)]
pub struct ResolvedPreimage {
    pub ptr: *mut u8,
    pub len: isize, // negative if not found
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_set_preimage_resolver(
    mach: *mut Machine,
    resolver: unsafe extern "C" fn(u64, *const u8) -> ResolvedPreimage,
) {
    (*mach).set_preimage_resolver(
        Arc::new(move |context: u64, hash: Bytes32| -> Option<CBytes> {
            let res = resolver(context, hash.as_ptr());
            if res.len < 0 {
                return None;
            }
            let data = CBytes::from_raw_parts(res.ptr, res.len as usize);
            let have_hash = Keccak256::digest(&data);
            if have_hash.as_slice() != *hash {
                panic!(
                    "Resolved incorrect data for hash {}: got {}",
                    hash,
                    hex::encode(data),
                );
            }
            Some(data)
        }) as PreimageResolver,
    );
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_set_context(mach: *mut Machine, context: u64) {
    (*mach).set_context(context);
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_hash(mach: *mut Machine) -> utils::Bytes32 {
    (*mach).hash()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_module_root(mach: *mut Machine) -> utils::Bytes32 {
    (*mach).get_modules_root()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_gen_proof(mach: *mut Machine) -> RustByteArray {
    let mut proof = (*mach).serialize_proof();
    let ret = RustByteArray {
        ptr: proof.as_mut_ptr(),
        len: proof.len(),
        capacity: proof.capacity(),
    };
    std::mem::forget(proof);
    ret
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_free_proof(proof: RustByteArray) {
    drop(Vec::from_raw_parts(proof.ptr, proof.len, proof.capacity))
}
