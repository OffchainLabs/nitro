// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    os::raw::{c_char, c_int},
    path::Path,
    ptr, slice,
    sync::{atomic, atomic::AtomicU8},
};

use arbutil::Bytes32;
use prover::{
    machine::{argument_data_to_inbox, get_empty_preimage_resolver, GlobalState, MachineStatus},
    Machine,
};
use static_assertions::const_assert_eq;

use crate::{
    c_strings::{c_string_to_string, err_to_c_string},
    CByteArray, RustBytes,
};

pub const ARBITRATOR_MACHINE_STATUS_RUNNING: u8 = 0;
pub const ARBITRATOR_MACHINE_STATUS_FINISHED: u8 = 1;
pub const ARBITRATOR_MACHINE_STATUS_ERRORED: u8 = 2;
pub const ARBITRATOR_MACHINE_STATUS_TOO_FAR: u8 = 3;

// Unfortunately, cbindgen doesn't support constants with non-literal values, so we assert that
// they're correct.
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

#[no_mangle]
pub unsafe extern "C" fn arbitrator_load_machine(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
    debug_chain: usize,
) -> *mut Machine {
    let debug_chain = debug_chain != 0;
    arbitrator_load_machine_impl(binary_path, library_paths, library_paths_size, debug_chain)
        .unwrap_or_else(|err| {
            eprintln!("Error loading binary: {err:?}");
            ptr::null_mut()
        })
}

unsafe fn arbitrator_load_machine_impl(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
    debug_chain: bool,
) -> eyre::Result<*mut Machine> {
    let binary_path = c_string_to_string(binary_path)?;
    let binary_path = Path::new(&binary_path);

    let mut libraries = vec![];
    for i in 0..library_paths_size {
        let path = c_string_to_string(*(library_paths.offset(i)))?;
        libraries.push(Path::new(&path).to_owned());
    }

    let mach = Machine::from_paths(
        &libraries,
        binary_path,
        true,
        true,
        debug_chain,
        debug_chain,
        Default::default(),
        Default::default(),
        get_empty_preimage_resolver(),
    )?;
    Ok(Box::into_raw(Box::new(mach)))
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_load_wavm_binary(binary_path: *const c_char) -> *mut Machine {
    let binary_path = match c_string_to_string(binary_path) {
        Ok(s) => s,
        Err(err) => {
            eprintln!("Error decoding binary path: {err}");
            return ptr::null_mut();
        }
    };
    let binary_path = Path::new(&binary_path);
    match Machine::new_from_wavm(binary_path) {
        Ok(mach) => Box::into_raw(Box::new(mach)),
        Err(err) => {
            eprintln!("Error loading binary: {err}");
            ptr::null_mut()
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_new_finished(gs: GlobalState) -> *mut Machine {
    Box::into_raw(Box::new(Machine::new_finished(gs)))
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
    ptr::null_mut()
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
        let data = data.as_slice().to_vec();
        mach.add_inbox_msg(identifier, index, data);
        0
    } else {
        eprintln!("Unknown inbox identifier: {inbox_identifier}");
        1
    }
}

/// Adds a user program to the machine's known set of wasms.
#[no_mangle]
pub unsafe extern "C" fn arbitrator_add_user_wasm(
    mach: *mut Machine,
    module: *const u8,
    module_len: usize,
    module_hash: *const Bytes32,
) {
    let module = slice::from_raw_parts(module, module_len);
    (*mach).add_stylus_module(*module_hash, module.to_owned());
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
                return ptr::null_mut();
            }
            if mach.next_instruction_is_host_io() {
                return ptr::null_mut();
            }
            match mach.step_n(1) {
                Ok(()) => {}
                Err(err) => return err_to_c_string(err),
            }
        }
    }
    ptr::null_mut()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_serialize_state(
    mach: *const Machine,
    path: *const c_char,
) -> c_int {
    let mach = &*mach;
    let res = c_string_to_string(path).and_then(|path| mach.serialize_state(path));
    if let Err(err) = res {
        eprintln!("Failed to serialize machine state: {err}");
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
    let res = c_string_to_string(path).and_then(|path| mach.deserialize_and_replace_state(path));
    if let Err(err) = res {
        eprintln!("Failed to deserialize machine state: {err}");
        1
    } else {
        0
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_get_num_steps(mach: *const Machine) -> u64 {
    (*mach).get_steps()
}

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

#[no_mangle]
pub unsafe extern "C" fn arbitrator_set_context(mach: *mut Machine, context: u64) {
    (*mach).set_context(context);
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_hash(mach: *mut Machine) -> Bytes32 {
    (*mach).hash()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_module_root(mach: *mut Machine) -> Bytes32 {
    (*mach).get_modules_root()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_gen_proof(mach: *mut Machine, out: *mut RustBytes) {
    (*out).write((*mach).serialize_proof());
}
