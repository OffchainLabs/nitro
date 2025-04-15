// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)] // TODO: require safety docs
#![no_std]

use caller_env::{self, wasip1_stub::Errno, GuestPtr};
use paste::paste;
use wee_alloc::WeeAlloc;

extern "C" {
    fn wavm_halt_and_set_finished() -> !;
}

#[global_allocator]
static ALLOC: WeeAlloc = WeeAlloc::INIT;

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
    ($(fn $func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty);*) => {
        paste! {
            $(
                #[no_mangle]
                pub unsafe extern "C" fn [<wasi_snapshot_preview1__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                    caller_env::wasip1_stub::$func_name(
                        &mut caller_env::static_caller::STATIC_MEM,
                        &mut caller_env::static_caller::STATIC_ENV,
                        $($arg_name),*
                    )
                }
            )*
        }
    };
}

wrap! {
    fn clock_time_get(
        clock_id: u32,
        precision: u64,
        time_ptr: GuestPtr
    ) -> Errno;

    fn random_get(buf: GuestPtr, len: u32) -> Errno;

    fn environ_sizes_get(length_ptr: GuestPtr, data_size_ptr: GuestPtr) -> Errno;

    fn fd_write(
        fd: u32,
        iovecs_ptr: GuestPtr,
        iovecs_len: u32,
        ret_ptr: GuestPtr
    ) -> Errno;

    fn environ_get(a: GuestPtr, b: GuestPtr) -> Errno;

    fn fd_close(fd: u32) -> Errno;
    fn fd_read(a: u32, b: u32, c: u32, d: u32) -> Errno;
    fn fd_readdir(
        fd: u32,
        a: u32,
        b: u32,
        c: u64,
        d: u32
    ) -> Errno;

    fn fd_sync(a: u32) -> Errno;

    fn fd_seek(
        fd: u32,
        offset: u64,
        whence: u8,
        filesize: u32
    ) -> Errno;

    fn fd_datasync(fd: u32) -> Errno;

    fn path_open(
        a: u32,
        b: u32,
        c: u32,
        d: u32,
        e: u32,
        f: u64,
        g: u64,
        h: u32,
        i: u32
    ) -> Errno;

    fn path_create_directory(
        a: u32,
        b: u32,
        c: u32
    ) -> Errno;

    fn path_remove_directory(
        a: u32,
        b: u32,
        c: u32
    ) -> Errno;

    fn path_readlink(
        a: u32,
        b: u32,
        c: u32,
        d: u32,
        e: u32,
        f: u32
    ) -> Errno;

    fn path_rename(
        a: u32,
        b: u32,
        c: u32,
        d: u32,
        e: u32,
        f: u32
    ) -> Errno;

    fn path_filestat_get(
        a: u32,
        b: u32,
        c: u32,
        d: u32,
        e: u32
    ) -> Errno;

    fn path_unlink_file(a: u32, b: u32, c: u32) -> Errno;

    fn fd_prestat_get(a: u32, b: u32) -> Errno;
    fn fd_prestat_dir_name(a: u32, b: u32, c: u32) -> Errno;

    fn fd_filestat_get(fd: u32, filestat: u32) -> Errno;
    fn fd_filestat_set_size(fd: u32, size: u64) -> Errno;

    fn fd_pread(
        fd: u32,
        a: u32,
        b: u32,
        c: u64,
        d: u32
    ) -> Errno;

    fn fd_pwrite(
        fd: u32,
        a: u32,
        b: u32,
        c: u64,
        d: u32
    ) -> Errno;

    fn sock_accept(fd: u32, a: u32, b: u32) -> Errno;
    fn sock_shutdown(a: u32, b: u32) -> Errno;

    fn sched_yield() -> Errno;

    fn args_sizes_get(
        length_ptr: GuestPtr,
        data_size_ptr: GuestPtr
    ) -> Errno;

    fn args_get(argv_buf: GuestPtr, data_buf: GuestPtr) -> Errno;

    fn fd_fdstat_get(a: u32, b: u32) -> Errno;
    fn fd_fdstat_set_flags(a: u32, b: u32) -> Errno;

    fn poll_oneoff(
        in_subs: GuestPtr,
        out_evt: GuestPtr,
        nsubscriptions: u32,
        nevents_ptr: GuestPtr
    ) -> Errno
}
