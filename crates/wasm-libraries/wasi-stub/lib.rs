// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)] // TODO: require safety docs
#![no_std]

use caller_env::{
    self, GuestPtr,
    static_caller::{StaticExecEnv, StaticMem},
    wasip1_stub::Errno,
};
use paste::paste;
use wee_alloc::WeeAlloc;

#[cfg(target_arch = "wasm32")]
unsafe extern "C" {
    fn wavm_halt_and_set_finished() -> !;
}

#[global_allocator]
static ALLOC: WeeAlloc = WeeAlloc::INIT;

#[cfg(target_arch = "wasm32")]
#[panic_handler]
unsafe fn panic(_: &core::panic::PanicInfo) -> ! {
    core::arch::wasm32::unreachable()
}

#[cfg(target_arch = "wasm32")]
#[unsafe(no_mangle)]
pub unsafe extern "C" fn wasi_snapshot_preview1__proc_exit(code: u32) -> ! {
    if code == 0 {
        unsafe { wavm_halt_and_set_finished() }
    } else {
        core::arch::wasm32::unreachable()
    }
}

macro_rules! wrap {
    ($(fn $func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty);*) => {
        paste! {
            $(
                #[unsafe(no_mangle)]
                pub unsafe extern "C" fn [<wasi_snapshot_preview1__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                    caller_env::wasip1_stub::$func_name(&mut StaticMem, $($arg_name),*)
                }
            )*
        }
    };
}

macro_rules! wrap_exec {
    ($(fn $func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty);*) => {
        paste! {
            $(
                #[unsafe(no_mangle)]
                pub unsafe extern "C" fn [<wasi_snapshot_preview1__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                    caller_env::wasip1_stub::$func_name(&mut StaticMem, &mut StaticExecEnv, $($arg_name),*)
                }
            )*
        }
    };
}

wrap_exec! {
    fn clock_time_get(a: u32, b: u64, c: GuestPtr) -> Errno;
    fn random_get(a: GuestPtr, b: u32) -> Errno;
    fn fd_write(a: u32, b: GuestPtr, c: u32, d: GuestPtr) -> Errno;
    fn poll_oneoff(a: GuestPtr, b: GuestPtr, c: u32, d: GuestPtr) -> Errno
}

wrap! {
    fn environ_sizes_get(a: GuestPtr, b: GuestPtr) -> Errno;
    fn environ_get(a: GuestPtr, b: GuestPtr) -> Errno;
    fn fd_close(a: u32) -> Errno;
    fn fd_read(a: u32, b: u32, c: u32, d: u32) -> Errno;
    fn fd_readdir(a: u32, b: u32, c: u32, d: u64, e: u32) -> Errno;
    fn fd_sync(a: u32) -> Errno;
    fn fd_seek(a: u32, b: u64, c: u8, d: u32) -> Errno;
    fn fd_datasync(a: u32) -> Errno;
    fn fd_fdstat_get(a: u32, b: u32) -> Errno;
    fn fd_fdstat_set_flags(a: u32, b: u32) -> Errno;
    fn fd_prestat_get(a: u32, b: u32) -> Errno;
    fn fd_prestat_dir_name(a: u32, b: u32, c: u32) -> Errno;
    fn fd_filestat_get(a: u32, b: u32) -> Errno;
    fn fd_filestat_set_size(a: u32, b: u64) -> Errno;
    fn fd_pread(a: u32, b: u32, c: u32, d: u64, e: u32) -> Errno;
    fn fd_pwrite(a: u32, b: u32, c: u32, d: u64, e: u32) -> Errno;
    fn path_open(a: u32, b: u32, c: u32, d: u32, e: u32, f: u64, g: u64, h: u32, i: u32) -> Errno;
    fn path_create_directory(a: u32, b: u32, c: u32) -> Errno;
    fn path_remove_directory(a: u32, b: u32, c: u32) -> Errno;
    fn path_readlink(a: u32, b: u32, c: u32, d: u32, e: u32, f: u32) -> Errno;
    fn path_rename(a: u32, b: u32, c: u32, d: u32, e: u32, f: u32) -> Errno;
    fn path_filestat_get(a: u32, b: u32, c: u32, d: u32, e: u32) -> Errno;
    fn path_unlink_file(a: u32, b: u32, c: u32) -> Errno;
    fn sock_accept(a: u32, b: u32, c: u32) -> Errno;
    fn sock_shutdown(a: u32, b: u32) -> Errno;
    fn sched_yield() -> Errno;
    fn args_sizes_get(a: GuestPtr, b: GuestPtr) -> Errno;
    fn args_get(a: GuestPtr, b: GuestPtr) -> Errno
}
