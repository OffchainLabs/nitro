// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_std]

use rand::RngCore;
use rand_pcg::Pcg32;

type Errno = u16;

type Uptr = usize;

const ERRNO_SUCCESS: Errno = 0;
const ERRNO_BADF: Errno = 8;
const ERRNO_INTVAL: Errno = 28;

#[allow(dead_code)]
extern "C" {
    fn wavm_caller_load8(ptr: Uptr) -> u8;
    fn wavm_caller_load32(ptr: Uptr) -> u32;
    fn wavm_caller_store8(ptr: Uptr, val: u8);
    fn wavm_caller_store32(ptr: Uptr, val: u32);
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
pub unsafe extern "C" fn wasi_snapshot_preview1__environ_sizes_get(
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Errno {
    wavm_caller_store32(length_ptr, 0);
    wavm_caller_store32(data_size_ptr, 0);
    ERRNO_SUCCESS
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_write(
    fd: usize,
    iovecs_ptr: Uptr,
    iovecs_len: usize,
    ret_ptr: Uptr,
) -> Errno {
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
pub unsafe extern "C" fn wasi_snapshot_preview1__environ_get(_: usize, _: usize) -> Errno {
    ERRNO_INTVAL
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_close(_: usize) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_read(
    _: usize,
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_readdir(
    _fd: usize,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_sync(
    _: u32,
) -> Errno {
    ERRNO_SUCCESS
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_seek(
    _fd: usize,
    _offset: u64,
    _whence: u8,
    _filesize: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_datasync(_fd: usize) -> Errno {
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
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_create_directory(
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_remove_directory(
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_readlink(
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_rename(
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_filestat_get(
    _: usize,
    _: usize,
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__path_unlink_file(
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_prestat_get(_: usize, _: usize) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_prestat_dir_name(
    _: usize,
    _: usize,
    _: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_filestat_get(
    _fd: usize,
    _filestat: usize,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_filestat_set_size(
    _fd: usize,
    _: u64,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_pread(
    _fd: usize,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_pwrite(
    _fd: usize,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__sock_accept(
    _fd: usize,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__sock_shutdown(
    _: usize,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__sched_yield() -> Errno {
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
    time: Uptr,
) -> Errno {
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
pub unsafe extern "C" fn wasi_snapshot_preview1__random_get(mut buf: usize, mut len: usize) -> Errno {
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

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__args_sizes_get(
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Errno {
    wavm_caller_store32(length_ptr, 1);
    wavm_caller_store32(data_size_ptr, 4);
    ERRNO_SUCCESS
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__args_get(
    argv_buf: Uptr, 
    data_buf: Uptr
) -> Errno {
    wavm_caller_store32(argv_buf, data_buf as u32);
    wavm_caller_store32(data_buf, 0x6E6962); // "bin\0"
    ERRNO_SUCCESS
}

#[no_mangle]
// we always simulate a timeout
pub unsafe extern "C" fn wasi_snapshot_preview1__poll_oneoff(in_subs: Uptr, out_evt: Uptr, nsubscriptions: usize, nevents_ptr: Uptr) -> Errno {
    const SUBSCRIPTION_SIZE: usize = 48;
    for i in 0..nsubscriptions {
        let subs_base = in_subs + (SUBSCRIPTION_SIZE * i);
        let subs_type = wavm_caller_load32(subs_base + 8);
        if subs_type != 0 {
            // not a clock subscription type
            continue
        }
        let user_data = wavm_caller_load32(subs_base);
        wavm_caller_store32(out_evt, user_data);
        wavm_caller_store32(out_evt + 8, 0);
        wavm_caller_store32(nevents_ptr, 1);
        return ERRNO_SUCCESS;
    }
    ERRNO_INTVAL
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_fdstat_get(_: usize, _: usize) -> Errno {
    ERRNO_INTVAL
}

#[no_mangle]
pub unsafe extern "C" fn wasi_snapshot_preview1__fd_fdstat_set_flags(_: usize, _: usize) -> Errno {
    ERRNO_INTVAL
}
