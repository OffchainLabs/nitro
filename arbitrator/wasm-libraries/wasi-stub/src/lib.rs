// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_std]

use rand::RngCore;
use rand_pcg::Pcg32;

const ERRNO_SUCCESS: u16 = 0;
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
    ERRNO_SUCCESS
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
    ERRNO_SUCCESS
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
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_seek(
    _fd: usize,
    _offset: u64,
    _whence: u8,
    _filesize: usize,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_datasync(_fd: usize) -> u16 {
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
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_filestat_get(
    _fd: usize,
    _filestat: usize,
) -> u16 {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__sched_yield() -> u16 {
    ERRNO_SUCCESS
}

// An increasing clock, measured in nanoseconds.
static mut TIME: u64 = 0;
// The amount of TIME advanced each check. Currently 10 milliseconds.
static TIME_INTERVAL: u64 = 10_000_000;

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__clock_time_get(
    _clock_id: usize,
    _precision: u64,
    time: usize,
) -> u16 {
    TIME += TIME_INTERVAL;
    wavm_caller_store32(time, TIME as u32);
    wavm_caller_store32(time + 4, (TIME >> 32) as u32);
    ERRNO_SUCCESS
}

static mut RNG: Option<Pcg32> = None;

unsafe fn get_rng<'a>() -> &'a mut Pcg32 {
    RNG.get_or_insert_with(|| Pcg32::new(0xcafef00dd15ea5e5, 0xa02bdbf7bb3c0a7))
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__random_get(mut buf: usize, mut len: usize) -> u16 {
    let rng = get_rng();
    while len >= 4 {
        wavm_caller_store32(buf, rng.next_u32());
        buf += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = rng.next_u32();
        for _ in 0..len {
            wavm_caller_store8(buf, rem as u8);
            buf += 1;
            rem >>= 8;
        }
    }
    ERRNO_SUCCESS
}
