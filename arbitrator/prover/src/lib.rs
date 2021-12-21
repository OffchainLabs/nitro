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
    machine::{InboxReaderFn, Machine},
};
use eyre::{bail, Result};
use fnv::FnvHashMap as HashMap;
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
            for (input, kind) in err.errors {
                eprintln!("Got {:?} while parsing {}", kind, hex::encode(&input[..64]));
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

#[repr(C)]
#[derive(Clone, Copy)]
pub struct CMultipleByteArrays {
    pub ptr: *const CByteArray,
    pub len: usize,
}

/// Note: the returned memory will not be freed by Arbitrator.
/// To indicate "not found", set len to non-zero and ptr to null.
/// context is copied from the machine, to be used by the function
type CInboxReaderFn = extern "C" fn(context: u64, inbox_idx: u64, seq_num: u64) -> CByteArray;

#[no_mangle]
pub unsafe extern "C" fn arbitrator_load_machine(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
    global_state: GlobalState,
    c_preimages: CMultipleByteArrays,
    c_inbox_reader: CInboxReaderFn,
) -> *mut Machine {
    match arbitrator_load_machine_impl(
        binary_path,
        library_paths,
        library_paths_size,
        global_state,
        c_preimages,
        c_inbox_reader,
    ) {
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
    global_state: GlobalState,
    c_preimages: CMultipleByteArrays,
    c_inbox_reader: CInboxReaderFn,
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

    let inbox_reader = Box::new(
        move |context: u64, inbox_idx: u64, seq_num: u64| -> Option<Vec<u8>> {
            unsafe {
                let res = c_inbox_reader(context, inbox_idx, seq_num);
                if res.len > 0 && res.ptr.is_null() {
                    None
                } else {
                    let slice = std::slice::from_raw_parts(res.ptr, res.len);
                    Some(slice.to_vec())
                }
            }
        },
    ) as InboxReaderFn;
    let mut preimages = HashMap::default();
    for i in 0..c_preimages.len {
        let c_bytes = *c_preimages.ptr.add(i);
        let slice = std::slice::from_raw_parts(c_bytes.ptr, c_bytes.len);
        let data = slice.to_vec();
        let hash = Keccak256::digest(&data);
        preimages.insert(hash.into(), data);
    }

    let mach = Machine::from_binary(
        libraries,
        main_mod,
        false,
        false,
        global_state,
        HashMap::default(),
        inbox_reader,
        preimages,
    );
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
pub unsafe extern "C" fn arbitrator_serialize_state(mach: *const Machine, path: *const c_char) -> c_int {
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
pub unsafe extern "C" fn arbitrator_deserialize_and_replace_state(mach: *mut Machine, path: *const c_char) -> c_int {
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
pub unsafe extern "C" fn arbitrator_set_inbox_reader_context(mach: *mut Machine, context: u64) {
    (*mach).set_inbox_reader_context(context);
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_add_preimages(
    mach: *mut Machine,
    c_preimages: CMultipleByteArrays,
) {
    for i in 0..c_preimages.len {
        let c_bytes = *c_preimages.ptr.add(i);
        let slice = std::slice::from_raw_parts(c_bytes.ptr, c_bytes.len);
        let data = slice.to_vec();
        let hash = Keccak256::digest(&data);
        (*mach).add_preimage(hash.into(), data)
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_hash(mach: *mut Machine) -> utils::Bytes32 {
    (*mach).hash()
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
