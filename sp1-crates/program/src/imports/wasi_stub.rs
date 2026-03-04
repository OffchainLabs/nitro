//! WASI stubs delegating to caller-env's shared implementation.

#![allow(clippy::too_many_arguments)]

use crate::{Escape, Ptr, state::{sp1_env, gp}, platform, replay::CustomEnvData};
use caller_env;
use wasmer::FunctionEnvMut;

pub fn proc_exit(mut ctx: FunctionEnvMut<CustomEnvData>, code: u32) {
    let (data, _store) = ctx.data_and_store_mut();

    if code == 0 {
        platform::print_string(
            1,
            format!(
                "Validation succeeds with hash {}",
                hex::encode(data.input().large_globals[0])
            )
            .as_bytes(),
        );
    }

    platform::exit(code);
}

pub fn args_sizes_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    argc: Ptr,
    argv_buf_size: Ptr,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::args_sizes_get(&mut mem, &mut state, gp(argc), gp(argv_buf_size)).0)
}

pub fn args_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    argv_buf: Ptr,
    data_buf: Ptr,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::args_get(&mut mem, &mut state, gp(argv_buf), gp(data_buf)).0)
}

pub fn environ_sizes_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    length_ptr: Ptr,
    data_size_ptr: Ptr,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::environ_sizes_get(&mut mem, &mut state, gp(length_ptr), gp(data_size_ptr)).0)
}

pub fn environ_get(mut ctx: FunctionEnvMut<CustomEnvData>, a: Ptr, b: Ptr) -> u16 {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    caller_env::wasip1_stub::environ_get(&mut mem, &mut state, gp(a), gp(b)).0
}

pub fn fd_write(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    fd: u32,
    iovecs_ptr: Ptr,
    iovecs_len: u32,
    ret_ptr: Ptr,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::fd_write(&mut mem, &mut state, fd, gp(iovecs_ptr), iovecs_len, gp(ret_ptr)).0)
}

pub fn clock_time_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    clock_id: u32,
    precision: u64,
    time_ptr: Ptr,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::clock_time_get(&mut mem, &mut state, clock_id, precision, gp(time_ptr)).0)
}

pub fn random_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    buf: Ptr,
    len: u32,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::random_get(&mut mem, &mut state, gp(buf), len).0)
}

pub fn poll_oneoff(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_subs: Ptr,
    out_evt: Ptr,
    num_subscriptions: u32,
    num_events_ptr: Ptr,
) -> Result<u16, Escape> {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    Ok(caller_env::wasip1_stub::poll_oneoff(&mut mem, &mut state, gp(in_subs), gp(out_evt), num_subscriptions, gp(num_events_ptr)).0)
}

pub fn fd_seek(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    fd: u32,
    offset: u64,
    whence: u32,
    filesize: u32,
) -> u16 {
    let (mut mem, mut state) = sp1_env(&mut ctx);
    caller_env::wasip1_stub::fd_seek(&mut mem, &mut state, fd, offset, whence as u8, filesize).0
}

macro_rules! wrap {
    ($(fn $func_name:ident ($($arg_name:ident : $arg_type:ty),*));* $(;)?) => {
        $(
            pub fn $func_name(mut ctx: FunctionEnvMut<CustomEnvData>, $($arg_name : $arg_type),*) -> u16 {
                let (mut mem, mut state) = sp1_env(&mut ctx);
                caller_env::wasip1_stub::$func_name(&mut mem, &mut state, $($arg_name),*).0
            }
        )*
    };
}

wrap! {
    fn fd_close(fd: u32);
    fn fd_read(a: u32, b: u32, c: u32, d: u32);
    fn fd_readdir(fd: u32, a: u32, b: u32, c: u64, d: u32);
    fn fd_sync(a: u32);
    fn fd_datasync(fd: u32);
    fn fd_fdstat_get(a: u32, b: u32);
    fn fd_fdstat_set_flags(a: u32, b: u32);
    fn fd_prestat_get(a: u32, b: u32);
    fn fd_prestat_dir_name(a: u32, b: u32, c: u32);
    fn fd_filestat_get(fd: u32, filestat: u32);
    fn fd_filestat_set_size(fd: u32, size: u64);
    fn fd_pread(fd: u32, a: u32, b: u32, c: u64, d: u32);
    fn fd_pwrite(fd: u32, a: u32, b: u32, c: u64, d: u32);
    fn path_open(a: u32, b: u32, c: u32, d: u32, e: u32, f: u64, g: u64, h: u32, i: u32);
    fn path_create_directory(a: u32, b: u32, c: u32);
    fn path_remove_directory(a: u32, b: u32, c: u32);
    fn path_readlink(a: u32, b: u32, c: u32, d: u32, e: u32, f: u32);
    fn path_rename(a: u32, b: u32, c: u32, d: u32, e: u32, f: u32);
    fn path_filestat_get(a: u32, b: u32, c: u32, d: u32, e: u32);
    fn path_unlink_file(a: u32, b: u32, c: u32);
    fn sock_accept(fd: u32, a: u32, b: u32);
    fn sock_shutdown(a: u32, b: u32);
    fn sched_yield()
}
