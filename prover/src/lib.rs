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

#[no_mangle]
pub unsafe extern "C" fn arbitrator_load_machine(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
    always_merkelize: bool,
) -> *mut Machine {
    match arbitrator_load_machine_impl(
        binary_path,
        library_paths,
        library_paths_size,
        always_merkelize,
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
    always_merkelize: bool,
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

    let global_state = crate::machine::GlobalState::default(); // TODO
    let inbox_reader = Box::new((|_, _| todo!("TODO: C inbox reader API")) as InboxReaderFn);
    let preimages = HashMap::default(); // TODO

    let mach = Machine::from_binary(
        libraries,
        main_mod,
        always_merkelize,
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
pub unsafe extern "C" fn arbitrator_step(mach: *mut Machine, num_steps: isize) {
    for _ in 0..num_steps {
        (*mach).step();
        if (*mach).is_halted() {
            break;
        }
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
    CString::from_raw(proof);
}
