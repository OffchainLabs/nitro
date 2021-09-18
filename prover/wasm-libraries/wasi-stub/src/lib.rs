#![no_std]

const ERRNO_BADF: u16 = 8;
const ERRNO_INTVAL: u16 = 28;

#[panic_handler]
unsafe fn panic(_: &core::panic::PanicInfo) -> ! {
    core::arch::wasm32::unreachable()
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__proc_exit(_: u32) {
    core::arch::wasm32::unreachable()
}

#[no_mangle]
pub unsafe extern "C" fn env__exit(_: u32) {
    core::arch::wasm32::unreachable()
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__environ_get(_: usize, _: usize) -> u16 {
    ERRNO_INTVAL
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_close(_: usize) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_read(_: usize, _: usize, _: usize, _: usize) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_open(_: usize, _: usize, _: usize, _: usize, _: usize, _: u64, _: u64, _: usize, _: usize) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_prestat_get(_: usize, _: usize) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_prestat_dir_name(_: usize, _: usize, _: usize) -> u16 {
    ERRNO_BADF
}
