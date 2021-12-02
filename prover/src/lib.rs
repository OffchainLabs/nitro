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
    ffi::{CStr, CString},
    fs::File,
    io::Read,
    os::raw::c_char,
    path::Path,
};

pub fn parse_binary(path: &Path) -> Result<WasmBinary> {
    let mut f = File::open(path)?;
    let mut buf = Vec::new();
    f.read_to_end(&mut buf)?;

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
pub struct CMultipleByteArrays {
    pub ptr: *const CByteArray,
    pub len: usize,
}

/// Note: the returned memory will not be freed by Arbitrator
type CInboxReaderFn = extern "C" fn(inbox_idx: u64, seq_num: u64) -> CByteArray;

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

    let inbox_reader = Box::new(move |inbox_idx: u64, seq_num: u64| -> Vec<u8> {
        unsafe {
            let res = c_inbox_reader(inbox_idx, seq_num);
            let slice = std::slice::from_raw_parts(res.ptr, res.len);
            slice.to_vec()
        }
    }) as InboxReaderFn;
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

#[no_mangle]
pub unsafe extern "C" fn arbitrator_step(mach: *mut Machine, num_steps: u64) {
    (*mach).step_n(num_steps.into());
}

#[repr(C)]
pub struct RustBytes {
    pub ptr: *mut u8,
    pub len: usize,
    pub capacity: usize,
}

/// The returned bytes must be freed with `arbitrator_free_bytes`
#[no_mangle]
pub unsafe extern "C" fn arbitrator_get_num_steps_be(mach: *const Machine) -> RustBytes {
    let mut v = (*mach).get_steps_bytes_be();
    let bytes = RustBytes {
        ptr: v.as_mut_ptr(),
        len: v.len(),
        capacity: v.capacity(),
    };
    std::mem::forget(v);
    bytes
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_free_bytes(bytes: RustBytes) {
    drop(Vec::from_raw_parts(bytes.ptr, bytes.len, bytes.capacity));
}

// C requires enums be represented as `int`s, so we need a new type for this :/
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
#[repr(C)]
pub enum CMachineStatus {
    Running,
    Finished,
    Errored,
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_get_status(mach: *const Machine) -> CMachineStatus {
    match (*mach).get_status() {
        MachineStatus::Running => CMachineStatus::Running,
        MachineStatus::Finished => CMachineStatus::Finished,
        MachineStatus::Errored => CMachineStatus::Errored,
    }
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_hash(mach: *mut Machine) -> utils::Bytes32 {
    (*mach).hash()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_gen_proof(mach: *mut Machine) -> *mut c_char {
    let proof = (*mach).serialize_proof();
    CString::new(hex::encode(proof))
        .expect("CString new failed")
        .into_raw()
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_free_proof(proof: *mut c_char) {
    drop(CString::from_raw(proof));
}
