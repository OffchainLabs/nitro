// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

pub mod binary;
mod host;
#[cfg(feature = "native")]
mod kzg;
pub mod machine;
/// cbindgen:ignore
pub mod memory;
pub mod merkle;
pub mod parse_input;
pub mod prepare;
mod print;
pub mod programs;
mod reinterpret;
pub mod utils;
pub mod value;
pub mod wavm;

#[cfg(test)]
mod test;

pub use machine::Machine;

use arbutil::{Bytes32, PreimageType};
use eyre::{Report, Result};
use lru::LruCache;
use machine::{
    argument_data_to_inbox, get_empty_preimage_resolver, GlobalState, MachineStatus,
    PreimageResolver,
};
use once_cell::sync::OnceCell;
use static_assertions::const_assert_eq;
use std::{
    ffi::CStr,
    marker::PhantomData,
    num::NonZeroUsize,
    os::raw::{c_char, c_int},
    path::Path,
    ptr, slice,
    sync::{
        atomic::{self, AtomicBool, AtomicU64, AtomicU8, Ordering},
        Arc, Mutex,
    },
};
use utils::CBytes;

lazy_static::lazy_static! {
    static ref BLOBHASH_PREIMAGE_CACHE: Mutex<LruCache<Bytes32, Arc<OnceCell<CBytes>>>> = Mutex::new(LruCache::new(NonZeroUsize::new(12).unwrap()));
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct CByteArray {
    pub ptr: *const u8,
    pub len: usize,
}

#[repr(C)]
pub struct RustSlice<'a> {
    pub ptr: *const u8,
    pub len: usize,
    pub phantom: PhantomData<&'a [u8]>,
}

impl<'a> RustSlice<'a> {
    pub fn new(slice: &'a [u8]) -> Self {
        if slice.is_empty() {
            return Self {
                ptr: ptr::null(),
                len: 0,
                phantom: PhantomData,
            };
        }
        Self {
            ptr: slice.as_ptr(),
            len: slice.len(),
            phantom: PhantomData,
        }
    }
}

#[repr(C)]
pub struct RustBytes {
    pub ptr: *mut u8,
    pub len: usize,
    pub cap: usize,
}

impl RustBytes {
    pub unsafe fn into_vec(self) -> Vec<u8> {
        Vec::from_raw_parts(self.ptr, self.len, self.cap)
    }

    pub unsafe fn write(&mut self, mut vec: Vec<u8>) {
        if vec.capacity() == 0 {
            *self = RustBytes {
                ptr: ptr::null_mut(),
                len: 0,
                cap: 0,
            };
            return;
        }
        self.ptr = vec.as_mut_ptr();
        self.len = vec.len();
        self.cap = vec.capacity();
        std::mem::forget(vec);
    }
}

/// Frees the vector. Does nothing when the vector is null.
///
/// # Safety
///
/// Must only be called once per vec.
#[no_mangle]
pub unsafe extern "C" fn free_rust_bytes(vec: RustBytes) {
    if !vec.ptr.is_null() {
        drop(vec.into_vec())
    }
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_load_machine(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
    debug_chain: usize,
) -> *mut Machine {
    let debug_chain = debug_chain != 0;
    match arbitrator_load_machine_impl(binary_path, library_paths, library_paths_size, debug_chain)
    {
        Ok(mach) => mach,
        Err(err) => {
            eprintln!("Error loading binary: {err:?}");
            ptr::null_mut()
        }
    }
}

unsafe fn arbitrator_load_machine_impl(
    binary_path: *const c_char,
    library_paths: *const *const c_char,
    library_paths_size: isize,
    debug_chain: bool,
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
        true,
        debug_chain,
        debug_chain,
        Default::default(),
        Default::default(),
        get_empty_preimage_resolver(),
    )?;
    let boxed = Box::new(mach);
    profiler_on_machine_created(&boxed);
    Ok(Box::into_raw(boxed))
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_load_wavm_binary(binary_path: *const c_char) -> *mut Machine {
    let binary_path = cstr_to_string(binary_path);
    let binary_path = Path::new(&binary_path);
    match Machine::new_from_wavm(binary_path) {
        Ok(mach) => {
            let boxed = Box::new(mach);
            profiler_on_machine_created(&boxed);
            Box::into_raw(boxed)
        }
        Err(err) => {
            eprintln!("Error loading binary: {err}");
            ptr::null_mut()
        }
    }
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_new_finished(gs: GlobalState) -> *mut Machine {
    let boxed = Box::new(Machine::new_finished(gs));
    profiler_on_machine_created(&boxed);
    Box::into_raw(boxed)
}

unsafe fn cstr_to_string(c_str: *const c_char) -> String {
    CStr::from_ptr(c_str).to_string_lossy().into_owned()
}

pub fn err_to_c_string(err: Report) -> *mut libc::c_char {
    str_to_c_string(&format!("{err:?}"))
}

/// Copies the str-data into a libc free-able C string
pub fn str_to_c_string(text: &str) -> *mut libc::c_char {
    unsafe {
        let buf = libc::malloc(text.len() + 1); // includes null-terminating byte
        if buf.is_null() {
            panic!("Failed to allocate memory for error string");
        }
        ptr::copy_nonoverlapping(text.as_ptr(), buf as *mut u8, text.len());
        *(buf as *mut u8).add(text.len()) = 0;
        buf as *mut libc::c_char
    }
}

static PROFILER_ENABLED: AtomicBool = AtomicBool::new(false);
static MACHINES_CREATED: AtomicU64 = AtomicU64::new(0);
static MACHINES_FREED: AtomicU64 = AtomicU64::new(0);
static MACHINES_LIVE: AtomicU64 = AtomicU64::new(0);
static MEMORY_BYTES_CURRENT: AtomicU64 = AtomicU64::new(0);
static MEMORY_BYTES_PEAK: AtomicU64 = AtomicU64::new(0);
static STYLUS_BYTES_CURRENT: AtomicU64 = AtomicU64::new(0);
static STYLUS_BYTES_PEAK: AtomicU64 = AtomicU64::new(0);
static INBOX_BYTES_CURRENT: AtomicU64 = AtomicU64::new(0);
static INBOX_ENTRIES_CURRENT: AtomicU64 = AtomicU64::new(0);
static LAST_DESTROY_STEPS: AtomicU64 = AtomicU64::new(0);
static LAST_DESTROY_STATUS: AtomicU64 = AtomicU64::new(0);
static LAST_DESTROY_MEMORY_BYTES: AtomicU64 = AtomicU64::new(0);
static LAST_DESTROY_STYLUS_BYTES: AtomicU64 = AtomicU64::new(0);
static LAST_DESTROY_INBOX_BYTES: AtomicU64 = AtomicU64::new(0);

#[repr(C)]
pub struct ArbitratorProfilerSnapshot {
    pub machines_created: u64,
    pub machines_freed: u64,
    pub machines_live: u64,
    pub memory_current_bytes: u64,
    pub memory_peak_bytes: u64,
    pub stylus_bytes_current: u64,
    pub stylus_bytes_peak: u64,
    pub inbox_bytes_current: u64,
    pub inbox_entries_current: u64,
    pub last_destroy_steps: u64,
    pub last_destroy_status: u64,
    pub last_destroy_memory_bytes: u64,
    pub last_destroy_stylus_bytes: u64,
    pub last_destroy_inbox_bytes: u64,
}

fn profiler_enabled() -> bool {
    PROFILER_ENABLED.load(Ordering::Relaxed)
}

fn update_peak(counter: &AtomicU64, candidate: u64) {
    let mut current = counter.load(Ordering::Relaxed);
    while candidate > current {
        match counter.compare_exchange_weak(
            current,
            candidate,
            Ordering::Relaxed,
            Ordering::Relaxed,
        ) {
            Ok(_) => return,
            Err(existing) => current = existing,
        }
    }
}

fn profiler_on_machine_created(machine: &Machine) {
    if !profiler_enabled() {
        return;
    }
    let telemetry = machine.telemetry();
    MACHINES_CREATED.fetch_add(1, Ordering::Relaxed);
    MACHINES_LIVE.fetch_add(1, Ordering::Relaxed);
    let memory_current = MEMORY_BYTES_CURRENT.fetch_add(telemetry.memory_bytes, Ordering::Relaxed)
        + telemetry.memory_bytes;
    update_peak(&MEMORY_BYTES_PEAK, memory_current);
    let stylus_current = STYLUS_BYTES_CURRENT
        .fetch_add(telemetry.stylus_module_bytes, Ordering::Relaxed)
        + telemetry.stylus_module_bytes;
    update_peak(&STYLUS_BYTES_PEAK, stylus_current);
    INBOX_BYTES_CURRENT.fetch_add(telemetry.inbox_bytes, Ordering::Relaxed);
    INBOX_ENTRIES_CURRENT.fetch_add(telemetry.inbox_entries, Ordering::Relaxed);
}

fn profiler_on_machine_destroy(machine: &Machine) {
    if !profiler_enabled() {
        return;
    }
    let telemetry = machine.telemetry();
    MACHINES_FREED.fetch_add(1, Ordering::Relaxed);
    MACHINES_LIVE.fetch_sub(1, Ordering::Relaxed);
    MEMORY_BYTES_CURRENT.fetch_sub(telemetry.memory_bytes, Ordering::Relaxed);
    STYLUS_BYTES_CURRENT.fetch_sub(telemetry.stylus_module_bytes, Ordering::Relaxed);
    INBOX_BYTES_CURRENT.fetch_sub(telemetry.inbox_bytes, Ordering::Relaxed);
    INBOX_ENTRIES_CURRENT.fetch_sub(telemetry.inbox_entries, Ordering::Relaxed);
    LAST_DESTROY_STEPS.store(telemetry.steps, Ordering::Relaxed);
    LAST_DESTROY_STATUS.store(u64::from(telemetry.status), Ordering::Relaxed);
    LAST_DESTROY_MEMORY_BYTES.store(telemetry.memory_bytes, Ordering::Relaxed);
    LAST_DESTROY_STYLUS_BYTES.store(telemetry.stylus_module_bytes, Ordering::Relaxed);
    LAST_DESTROY_INBOX_BYTES.store(telemetry.inbox_bytes, Ordering::Relaxed);
}

#[no_mangle]
pub extern "C" fn arbitrator_profiler_set_enabled(enable: bool) {
    PROFILER_ENABLED.store(enable, Ordering::Relaxed);
}

#[no_mangle]
pub unsafe extern "C" fn arbitrator_profiler_snapshot(out: *mut ArbitratorProfilerSnapshot) {
    if out.is_null() {
        return;
    }
    let snapshot = ArbitratorProfilerSnapshot {
        machines_created: MACHINES_CREATED.load(Ordering::Relaxed),
        machines_freed: MACHINES_FREED.load(Ordering::Relaxed),
        machines_live: MACHINES_LIVE.load(Ordering::Relaxed),
        memory_current_bytes: MEMORY_BYTES_CURRENT.load(Ordering::Relaxed),
        memory_peak_bytes: MEMORY_BYTES_PEAK.load(Ordering::Relaxed),
        stylus_bytes_current: STYLUS_BYTES_CURRENT.load(Ordering::Relaxed),
        stylus_bytes_peak: STYLUS_BYTES_PEAK.load(Ordering::Relaxed),
        inbox_bytes_current: INBOX_BYTES_CURRENT.load(Ordering::Relaxed),
        inbox_entries_current: INBOX_ENTRIES_CURRENT.load(Ordering::Relaxed),
        last_destroy_steps: LAST_DESTROY_STEPS.load(Ordering::Relaxed),
        last_destroy_status: LAST_DESTROY_STATUS.load(Ordering::Relaxed),
        last_destroy_memory_bytes: LAST_DESTROY_MEMORY_BYTES.load(Ordering::Relaxed),
        last_destroy_stylus_bytes: LAST_DESTROY_STYLUS_BYTES.load(Ordering::Relaxed),
        last_destroy_inbox_bytes: LAST_DESTROY_INBOX_BYTES.load(Ordering::Relaxed),
    };
    out.write(snapshot);
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_free_machine(mach: *mut Machine) {
    if mach.is_null() {
        return;
    }
    let boxed = Box::from_raw(mach);
    profiler_on_machine_destroy(&boxed);
    drop(boxed);
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_clone_machine(mach: *mut Machine) -> *mut Machine {
    let new_mach = (*mach).clone();
    let boxed = Box::new(new_mach);
    profiler_on_machine_created(&boxed);
    Box::into_raw(boxed)
}

/// Go doesn't have this functionality builtin for whatever reason. Uses relaxed ordering.
#[no_mangle]
pub unsafe extern "C" fn atomic_u8_store(ptr: *mut u8, contents: u8) {
    (*(ptr as *mut AtomicU8)).store(contents, atomic::Ordering::Relaxed);
}

/// Runs the machine while the condition variable is zero. May return early if num_steps is hit.
/// Returns a c string error (freeable with libc's free) on error, or nullptr on success.
#[no_mangle]
#[cfg(feature = "native")]
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
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_add_inbox_message(
    mach: *mut Machine,
    inbox_identifier: u64,
    index: u64,
    data: CByteArray,
) -> c_int {
    let mach = &mut *mach;
    if let Some(identifier) = argument_data_to_inbox(inbox_identifier) {
        let slice = slice::from_raw_parts(data.ptr, data.len);
        let data = slice.to_vec();
        mach.add_inbox_msg(identifier, index, data);
        0
    } else {
        1
    }
}

/// Adds a user program to the machine's known set of wasms.
#[no_mangle]
#[cfg(feature = "native")]
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
#[cfg(feature = "native")]
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
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_serialize_state(
    mach: *const Machine,
    path: *const c_char,
) -> c_int {
    let mach = &*mach;
    let res = CStr::from_ptr(path)
        .to_str()
        .map_err(Report::from)
        .and_then(|path| mach.serialize_state(path));
    if let Err(err) = res {
        eprintln!("Failed to serialize machine state: {err}");
        1
    } else {
        0
    }
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_deserialize_and_replace_state(
    mach: *mut Machine,
    path: *const c_char,
) -> c_int {
    let mach = &mut *mach;
    let res = CStr::from_ptr(path)
        .to_str()
        .map_err(Report::from)
        .and_then(|path| mach.deserialize_and_replace_state(path));
    if let Err(err) = res {
        eprintln!("Failed to deserialize machine state: {err}");
        1
    } else {
        0
    }
}

#[no_mangle]
#[cfg(feature = "native")]
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
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_get_status(mach: *const Machine) -> u8 {
    (*mach).get_status() as u8
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_global_state(mach: *mut Machine) -> GlobalState {
    (*mach).get_global_state()
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_set_global_state(mach: *mut Machine, gs: GlobalState) {
    (*mach).set_global_state(gs);
}

#[repr(C)]
pub struct ResolvedPreimage {
    pub ptr: *mut u8,
    pub len: isize, // negative if not found
}

#[cfg(feature = "native")]
unsafe fn handle_preimage_resolution(
    context: u64,
    ty: PreimageType,
    hash: Bytes32,
    resolver: unsafe extern "C" fn(u64, u8, *const u8) -> ResolvedPreimage,
) -> Option<CBytes> {
    let res = resolver(context, ty.into(), hash.as_ptr());
    if res.len < 0 {
        return None;
    }
    let data = CBytes::from_raw_parts(res.ptr, res.len as usize);

    // Hash may not have a direct link to the data for DACertificate
    if ty == PreimageType::DACertificate {
        return Some(data);
    }

    // Check if preimage rehashes to the provided hash
    match crate::utils::hash_preimage(&data, ty) {
        Ok(have_hash) if have_hash.as_slice() == *hash => {}
        Ok(got_hash) => panic!(
            "Resolved incorrect data for hash {} (rehashed to {})",
            hash,
            Bytes32(got_hash),
        ),
        Err(err) => panic!("Failed to hash preimage from resolver (expecting hash {hash}): {err}",),
    }
    Some(data)
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_set_preimage_resolver(
    mach: *mut Machine,
    resolver: unsafe extern "C" fn(u64, u8, *const u8) -> ResolvedPreimage,
) {
    (*mach).set_preimage_resolver(Arc::new(
        move |context: u64, ty: PreimageType, hash: Bytes32| -> Option<CBytes> {
            if ty == PreimageType::EthVersionedHash {
                let cache: Arc<OnceCell<CBytes>> = {
                    let mut locked = BLOBHASH_PREIMAGE_CACHE.lock().unwrap();
                    locked.get_or_insert(hash, Default::default).clone()
                };
                return cache
                    .get_or_try_init(|| {
                        handle_preimage_resolution(context, ty, hash, resolver).ok_or(())
                    })
                    .ok()
                    .cloned();
            }
            handle_preimage_resolution(context, ty, hash, resolver)
        },
    ) as PreimageResolver);
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_set_context(mach: *mut Machine, context: u64) {
    (*mach).set_context(context);
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_hash(mach: *mut Machine) -> Bytes32 {
    (*mach).hash()
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_module_root(mach: *mut Machine) -> Bytes32 {
    (*mach).get_modules_root()
}

#[no_mangle]
#[cfg(feature = "native")]
pub unsafe extern "C" fn arbitrator_gen_proof(mach: *mut Machine, out: *mut RustBytes) {
    (*out).write((*mach).serialize_proof());
}
