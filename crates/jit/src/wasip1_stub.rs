// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::too_many_arguments)]

use crate::caller_env::{JitEnv, JitExecEnv};
use crate::machine::{Escape, WasmEnvMut};
use caller_env::{self, wasip1_stub::Errno, GuestPtr};

pub fn proc_exit(mut _env: WasmEnvMut, code: u32) -> Result<(), Escape> {
    Err(Escape::Exit(code))
}

macro_rules! wrap {
    (fn $func_name:ident ($($arg_name:ident : $arg_type:ty),* $(,)?)) => {
        pub fn $func_name(mut src: WasmEnvMut, $($arg_name : $arg_type),*) -> Result<Errno, Escape> {
            let (mut mem, wenv) = src.jit_env();
            Ok(caller_env::wasip1_stub::$func_name(&mut mem, &mut JitExecEnv { wenv }, $($arg_name),*))
        }
    };
}

wrap!(fn clock_time_get(clock_id: u32, precision: u64, time_ptr: GuestPtr));
wrap!(fn random_get(buf: GuestPtr, len: u32));
wrap!(fn environ_get(a: GuestPtr, b: GuestPtr));
wrap!(fn environ_sizes_get(length_ptr: GuestPtr, data_size_ptr: GuestPtr));
wrap!(fn fd_read(a: u32, b: u32, c: u32, d: u32));
wrap!(fn fd_close(fd: u32));
wrap!(fn fd_write(fd: u32, iovecs_ptr: GuestPtr, iovecs_len: u32, ret_ptr: GuestPtr));
wrap!(fn fd_readdir(fd: u32, a: u32, b: u32, c: u64, d: u32));
wrap!(fn fd_sync(a: u32));
wrap!(fn fd_seek(fd: u32, offset: u64, whence: u8, filesize: u32));
wrap!(fn fd_datasync(_fd: u32));
wrap!(fn path_open(a: u32, b: u32, c: u32, d: u32, e: u32, f: u64, g: u64, h: u32, i: u32));
wrap!(fn path_create_directory(a: u32, b: u32, c: u32));
wrap!(fn path_remove_directory(a: u32, b: u32, c: u32));
wrap!(fn path_readlink(a: u32, b: u32, c: u32, d: u32, e: u32, f: u32));
wrap!(fn path_rename(a: u32, b: u32, c: u32, d: u32, e: u32, f: u32));
wrap!(fn path_filestat_get(a: u32, b: u32, c: u32, d: u32, e: u32));
wrap!(fn path_unlink_file(a: u32, b: u32, c: u32));
wrap!(fn fd_prestat_get(a: u32, b: u32));
wrap!(fn fd_prestat_dir_name(a: u32, b: u32, c: u32));
wrap!(fn fd_filestat_get(fd: u32, _filestat: u32));
wrap!(fn fd_filestat_set_size(fd: u32, size: u64));
wrap!(fn fd_pread(fd: u32, a: u32, b: u32, c: u64, d: u32));
wrap!(fn fd_pwrite(fd: u32, a: u32, b: u32, c: u64, d: u32));
wrap!(fn sock_accept(_fd: u32, a: u32, b: u32));
wrap!(fn sock_shutdown(a: u32, b: u32));
wrap!(fn sched_yield());
wrap!(fn args_sizes_get(length_ptr: GuestPtr, data_size_ptr: GuestPtr));
wrap!(fn args_get(argv_buf: GuestPtr, data_buf: GuestPtr));
wrap!(fn fd_fdstat_get(a: u32, b: u32));
wrap!(fn fd_fdstat_set_flags(a: u32, b: u32));
wrap!(fn poll_oneoff(in_subs: GuestPtr, out_evt: GuestPtr, nsubscriptions: u32, nevents_ptr: GuestPtr));
