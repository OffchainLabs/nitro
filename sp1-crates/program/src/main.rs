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
