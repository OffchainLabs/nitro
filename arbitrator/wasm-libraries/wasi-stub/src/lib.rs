// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_std]

use rand::RngCore;
use paste::paste;
use callerenv::{
    self,
    wasip1_stub::{Errno, Uptr},
    CallerEnv,
};
#[allow(dead_code)]
extern "C" {
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

macro_rules! wrap {
    ($func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty) => {
        paste! {
            #[no_mangle]
            pub unsafe extern "C" fn [<wasi_snapshot_preview1__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                callerenv::wasip1_stub::$func_name(&mut callerenv::static_caller::STATIC_CALLER, $($arg_name),*)
            }
        }
    };
}

wrap!(clock_time_get(
    clock_id: u32,
    precision: u64,
    time_ptr: Uptr
) -> Errno);

wrap!(random_get(buf: Uptr, len: u32) -> Errno);

wrap!(environ_sizes_get(length_ptr: Uptr, data_size_ptr: Uptr) -> Errno);
wrap!(fd_write(
    fd: u32,
    iovecs_ptr: Uptr,
    iovecs_len: u32,
    ret_ptr: Uptr
) -> Errno);
wrap!(environ_get(a: u32, b: u32) -> Errno);
wrap!(fd_close(fd: u32) -> Errno);
wrap!(fd_read(a: u32, b: u32, c: u32, d: u32) -> Errno);
wrap!(fd_readdir(
    fd: u32,
    a: u32,
    b: u32,
    c: u64,
    d: u32
) -> Errno);

wrap!(fd_sync(a: u32) -> Errno);

wrap!(fd_seek(
    _fd: u32,
    _offset: u64,
    _whence: u8,
    _filesize: u32
) -> Errno);

wrap!(fd_datasync(_fd: u32) -> Errno);

wrap!(path_open(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32,
    f: u64,
    g: u64,
    h: u32,
    i: u32
) -> Errno);

wrap!(path_create_directory(
    a: u32,
    b: u32,
    c: u32
) -> Errno);

wrap!(path_remove_directory(
    a: u32,
    b: u32,
    c: u32
) -> Errno);

wrap!(path_readlink(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32,
    f: u32
) -> Errno);

wrap!(path_rename(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32,
    f: u32
) -> Errno);

wrap!(path_filestat_get(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32
) -> Errno);

wrap!(path_unlink_file(a: u32, b: u32, c: u32) -> Errno);

wrap!(fd_prestat_get(a: u32, b: u32) -> Errno);

wrap!(fd_prestat_dir_name(a: u32, b: u32, c: u32) -> Errno);

wrap!(fd_filestat_get(_fd: u32, _filestat: u32) -> Errno);

wrap!(fd_filestat_set_size(_fd: u32, size: u64) -> Errno);

wrap!(fd_pread(
    _fd: u32,
    _a: u32,
    _b: u32,
    _c: u64,
    _d: u32
) -> Errno);

wrap!(fd_pwrite(
    _fd: u32,
    _a: u32,
    _b: u32,
    _c: u64,
    _d: u32
) -> Errno);

wrap!(sock_accept(_fd: u32, a: u32, b: u32) -> Errno);

wrap!(sock_shutdown(a: u32, b: u32) -> Errno);

wrap!(sched_yield() -> Errno);

wrap!(args_sizes_get(
    length_ptr: Uptr,
    data_size_ptr: Uptr
) -> Errno);

wrap!(args_get(argv_buf: Uptr, data_buf: Uptr) -> Errno);

wrap!(poll_oneoff(
    in_subs: Uptr,
    out_evt: Uptr,
    nsubscriptions: u32,
    nevents_ptr: Uptr
) -> Errno);

wrap!(fd_fdstat_get(a: u32, b: u32) -> Errno);

wrap!(fd_fdstat_set_flags(a: u32, b: u32) -> Errno);
