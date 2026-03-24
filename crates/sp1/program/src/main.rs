#![cfg_attr(target_os = "zkvm", no_main)]

#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

fn main() {
    // We are loading replay.wasmu object here. After initializing, it is
    // not needed.
    let sp1_zkvm::ReadVecResult { ptr, len, .. } = sp1_zkvm::read_vec_raw();
    assert!(!ptr.is_null());
    // SAFETY: ptr must not be deallocated
    let s: &'static [u8] = unsafe { std::slice::from_raw_parts(ptr, len) };
    let metadata = bytes::Bytes::from_static(s);

    program::run(metadata);
}

// Those are referenced by wasmer runtimes, but are never invoked
#[unsafe(no_mangle)]
pub extern "C" fn __negdf2(_x: f64) -> f64 {
    todo!()
}

#[unsafe(no_mangle)]
pub extern "C" fn __negsf2(_x: f32) -> f32 {
    todo!()
}

// The following code provides adapter functions, so the brotli implementation
// in C can rely on Rust for memory management.

// Use `alloc::` instead of `std::` if you are in a `#![no_std]` environment.
use std::alloc::{Layout, alloc, alloc_zeroed, dealloc, realloc as rust_realloc};
use std::ptr;

// Alignment and header size (use 16 for 64-bit systems or SIMD requirements)
const ALIGN: usize = 8;
const HEADER_SIZE: usize = 8;

/// void* malloc(size_t size)
#[unsafe(no_mangle)]
pub unsafe extern "C" fn malloc(size: usize) -> *mut u8 {
    if size == 0 {
        return ptr::null_mut();
    }

    let total_size = size + HEADER_SIZE;
    let layout = Layout::from_size_align_unchecked(total_size, ALIGN);
    let ptr = alloc(layout);

    if ptr.is_null() {
        return ptr::null_mut();
    }

    *(ptr as *mut usize) = size;
    ptr.add(HEADER_SIZE)
}

/// void* calloc(size_t nmemb, size_t size)
#[unsafe(no_mangle)]
pub unsafe extern "C" fn calloc(nmemb: usize, size: usize) -> *mut u8 {
    let req_size = match nmemb.checked_mul(size) {
        Some(s) => s,
        None => return ptr::null_mut(),
    };

    if req_size == 0 {
        return ptr::null_mut();
    }

    let total_size = req_size + HEADER_SIZE;
    let layout = Layout::from_size_align_unchecked(total_size, ALIGN);
    let ptr = alloc_zeroed(layout);

    if ptr.is_null() {
        return ptr::null_mut();
    }

    *(ptr as *mut usize) = req_size;
    ptr.add(HEADER_SIZE)
}

/// void* realloc(void* ptr, size_t size)
#[unsafe(no_mangle)]
pub unsafe extern "C" fn realloc(ptr: *mut u8, size: usize) -> *mut u8 {
    // C standard: realloc(NULL, size) is identical to malloc(size)
    if ptr.is_null() {
        return malloc(size);
    }

    // C standard: realloc(ptr, 0) is identical to free(ptr)
    if size == 0 {
        free(ptr);
        return ptr::null_mut();
    }

    let header_ptr = ptr.sub(HEADER_SIZE);
    let old_size = *(header_ptr as *const usize);

    let old_total_size = old_size + HEADER_SIZE;
    let new_total_size = size + HEADER_SIZE;

    let layout = Layout::from_size_align_unchecked(old_total_size, ALIGN);

    let new_header_ptr = rust_realloc(header_ptr, layout, new_total_size);
    if new_header_ptr.is_null() {
        return ptr::null_mut(); // Original block is left untouched on failure
    }

    // Update the header with the new requested size
    *(new_header_ptr as *mut usize) = size;

    new_header_ptr.add(HEADER_SIZE)
}

/// void free(void* ptr)
#[unsafe(no_mangle)]
pub unsafe extern "C" fn free(ptr: *mut u8) {
    if ptr.is_null() {
        return;
    }

    let header_ptr = ptr.sub(HEADER_SIZE);
    let size = *(header_ptr as *const usize);

    let total_size = size + HEADER_SIZE;
    let layout = Layout::from_size_align_unchecked(total_size, ALIGN);

    dealloc(header_ptr, layout);
}

/// void exit(int status)
#[unsafe(no_mangle)]
pub extern "C" fn exit(status: i32) -> ! {
    // For embedded no_std, you will likely replace this with a panic
    // or a hardware-specific halt/reset routine.
    std::process::exit(status);
}
