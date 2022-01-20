pub mod binary;
mod host;
pub mod machine;
/// cbindgen:ignore
mod memory;
mod merkle;
mod reinterpret;
pub mod utils;
mod value;
pub mod wavm;

use crate::{
    binary::WasmBinary,
    machine::{argument_data_to_inbox, Machine},
};
use eyre::{bail, Result};
use machine::{GlobalState, MachineStatus};
use sha3::{Digest, Keccak256};
use std::{
    ffi::CStr,
    fs::File,
    io::Read,
    os::raw::{c_char, c_int},
    path::Path,
    process::Command,
    sync::atomic::{self, AtomicU8},
};

pub fn parse_binary(path: &Path) -> Result<WasmBinary> {
    let mut f = File::open(path)?;
    let mut buf = Vec::new();
    f.read_to_end(&mut buf)?;

    let mut cmd = Command::new("wasm-validate");
    if path.starts_with("-") {
        // Escape the path and ensure it isn't treated as a flag.
        // Unfortunately, older versions of wasm-validate don't support this,
        // so we only pass in this option if the path looks like a flag.
        cmd.arg("--");
    }
    let status = cmd.arg(path).status()?;
    if !status.success() {
        bail!("failed to validate WASM binary at {:?}", path);
    }

    let bin = match binary::parse(&buf) {
        Ok(bin) => bin,
        Err(err) => {
            eprintln!("Parsing error:");
            for (mut input, kind) in err.errors {
                if input.len() > 64 {
                    input = &input[..64];
                }
                eprintln!("Got {:?} while parsing {}", kind, hex::encode(input));
            }
            bail!("failed to parse binary");
        }
    };

    Ok(bin)
}

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
    let main_mod = {
        let binary_path = cstr_to_string(binary_path);
        let binary_path = Path::new(&binary_path);
        parse_binary(binary_path)?
    };

    let mut libraries = Vec::new();
    for i in 0..library_paths_size {
        let library_path = cstr_to_string(*(library_paths.offset(i)));
        let library_path = Path::new(&library_path);
        libraries.push(parse_binary(library_path)?);
    }

    let mach = Machine::from_binary(
        libraries,
        main_mod,
        false,
        false,
        Default::default(),
        Default::default(),
        Default::default(),
    )?;
    Ok(Box::into_raw(Box::new(mach)))
}

unsafe fn cstr_to_string(c_str: *const c_char) -> String {
    CStr::from_ptr(c_str).to_string_lossy().into_owned()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_free_machine(mach: *mut Machine) {
    Box::from_raw(mach);
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

/// Runs the machine while the condition variable is zero. May return early if num_steps is hit.
#[no_mangle]
pub unsafe extern "C" fn arbitrator_step(mach: *mut Machine, num_steps: u64, condition: *const u8) {
    let mach = &mut *mach;
    let condition = &*(condition as *const AtomicU8);
    let mut remaining_steps = num_steps;
    while condition.load(atomic::Ordering::Relaxed) == 0 {
        if remaining_steps == 0 || mach.is_halted() {
            break;
        }
        let stepping = std::cmp::min(remaining_steps, 1_000_000);
        mach.step_n(stepping);
        remaining_steps -= stepping;
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_add_inbox_message(
    mach: *mut Machine,
    inbox_identifier: u64,
    index: u64,
    data: CByteArray,
) -> c_int {
    let mach = &mut *mach;
    if let Ok(identifier) = argument_data_to_inbox(inbox_identifier) {
        let slice = std::slice::from_raw_parts(data.ptr, data.len);
        let data = slice.to_vec();
        mach.add_inbox_msg(identifier, index, data);
        0
    } else {
        1
    }
}

/// Like arbitrator_step, but stops early if it hits a host io operation.
#[no_mangle]
pub unsafe extern "C" fn arbitrator_step_until_host_io(mach: *mut Machine, condition: *const u8) {
    let mach = &mut *mach;
    let condition = &*(condition as *const AtomicU8);
    while condition.load(atomic::Ordering::Relaxed) == 0 {
        for _ in 0..1_000_000 {
            if mach.is_halted() {
                return;
            }
            if mach
                .get_next_instruction()
                .map(|i| i.opcode.is_host_io())
                .unwrap_or(true)
            {
                return;
            }
            mach.step();
        }
    }
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

// C requires enums be represented as `int`s, so we need a new type for this :/
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
#[repr(C)]
pub enum CMachineStatus {
    Running,
    Finished,
    Errored,
    TooFar,
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_get_status(mach: *const Machine) -> CMachineStatus {
    match (*mach).get_status() {
        MachineStatus::Running => CMachineStatus::Running,
        MachineStatus::Finished => CMachineStatus::Finished,
        MachineStatus::Errored => CMachineStatus::Errored,
        MachineStatus::TooFar => CMachineStatus::TooFar,
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_global_state(mach: *mut Machine) -> GlobalState {
    return (*mach).get_global_state();
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_set_global_state(mach: *mut Machine, gs: GlobalState) {
    (*mach).set_global_state(gs);
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_add_preimage(
    mach: *mut Machine,
    c_preimage: CByteArray,
) -> c_int {
    let slice = std::slice::from_raw_parts(c_preimage.ptr, c_preimage.len);
    let data = slice.to_vec();
    let hash = Keccak256::digest(&data);
    (*mach).add_preimage(hash.into(), data);
    0
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
