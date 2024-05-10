#![no_std]

const ERRNO_BADF: u16 = 8;
const ERRNO_INTVAL: u16 = 28;

#[allow(dead_code)]
extern "C" {
    fn wavm_caller_load8(ptr: usize) -> u8;
    fn wavm_caller_load32(ptr: usize) -> u32;
    fn wavm_caller_store8(ptr: usize, val: u8);
    fn wavm_caller_store32(ptr: usize, val: u32);
    fn wavm_halt_and_set_finished() -> !;
}

#[panic_handler]
unsafe fn panic(_: &core::panic::PanicInfo) -> ! {
    core::arch::wasm32::unreachable()
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__proc_exit(code: u32) -> ! {
    if code == 0 {
        wavm_halt_and_set_finished()
    } else {
        core::arch::wasm32::unreachable()
    }
}

#[no_mangle]
pub unsafe extern "C" fn env__exit(code: u32) {
    if code == 0 {
        wavm_halt_and_set_finished()
    } else {
        core::arch::wasm32::unreachable()
    }
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__environ_sizes_get(
    length_ptr: usize,
    data_size_ptr: usize,
) -> u16 {
    wavm_caller_store32(length_ptr, 0);
    wavm_caller_store32(data_size_ptr, 0);
    0
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_write(
    fd: usize,
    iovecs_ptr: usize,
    iovecs_len: usize,
    ret_ptr: usize,
) -> u16 {
    if fd != 1 && fd != 2 {
        return ERRNO_BADF;
    }
    let mut size = 0;
    for i in 0..iovecs_len {
        let ptr = iovecs_ptr + i * 8;
        size += wavm_caller_load32(ptr + 4);
    }
    wavm_caller_store32(ret_ptr, size);
    0
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
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_read(
    _: usize,
    _: usize,
    _: usize,
    _: usize,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_open(
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: u64,
    _: u64,
    _: usize,
    _: usize,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_prestat_get(_: usize, _: usize) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_prestat_dir_name(
    _: usize,
    _: usize,
    _: usize,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__sched_yield() -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__random_get(_: i32, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__clock_time_get(_: i32, _: i64, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_fdstat_get(_: i32, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_fdstat_set_flags(_: i32, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_filestat_get(_: i32, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_seek(_: i32, _: i64, _: i32, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_create_directory(
    _: i32,
    _: i32,
    _: i32,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_filestat_get(
    _: i32,
    _: i32,
    _: i32,
    _: i32,
    _: i32,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_rename(
    _: i32,
    _: i32,
    _: i32,
    _: i32,
    _: i32,
    _: i32,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_unlink_file(_: i32, _: i32, _: i32) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__sock_accept(_: i32, _: i32, _: i32) -> u16 {
    ERRNO_BADF
}
