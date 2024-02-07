// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::callerenv::CallerEnv;
use crate::machine::{Escape, WasmEnvMut};
use rand::RngCore;

type Errno = u16;

type Uptr = u32;

const ERRNO_SUCCESS: Errno = 0;
const ERRNO_BADF: Errno = 8;
const ERRNO_INTVAL: Errno = 28;

pub fn proc_exit(mut _env: WasmEnvMut, code: u32) -> Result<(), Escape> {
    Err(Escape::Exit(code))
}

pub fn environ_sizes_get(
    mut env: WasmEnvMut,
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);

    caller_env.caller_write_u32(length_ptr, 0);
    caller_env.caller_write_u32(data_size_ptr, 0);
    Ok(ERRNO_SUCCESS)
}

pub fn fd_write(
    mut env: WasmEnvMut,
    fd: u32,
    iovecs_ptr: Uptr,
    iovecs_len: u32,
    ret_ptr: Uptr,
) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);

    if fd != 1 && fd != 2 {
        return Ok(ERRNO_BADF);
    }
    let mut size = 0;
    for i in 0..iovecs_len {
        let ptr = iovecs_ptr + i * 8;
        let iovec = caller_env.caller_read_u32(ptr);
        let len = caller_env.caller_read_u32(ptr + 4);
        let data = caller_env.caller_read_string(iovec, len);
        eprintln!("JIT: WASM says [{fd}]: {data}");
        size += len;
    }
    caller_env.caller_write_u32(ret_ptr, size);
    Ok(ERRNO_SUCCESS)
}

pub fn environ_get(mut _env: WasmEnvMut, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_INTVAL)
}

pub fn fd_close(mut _env: WasmEnvMut, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_read(mut _env: WasmEnvMut, _: u32, _: u32, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_readdir(
    mut _env: WasmEnvMut,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_sync(mut _env: WasmEnvMut, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_SUCCESS)
}

pub fn fd_seek(
    mut _env: WasmEnvMut,
    _fd: u32,
    _offset: u64,
    _whence: u8,
    _filesize: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_datasync(mut _env: WasmEnvMut, _fd: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_open(
    mut _env: WasmEnvMut,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u64,
    _: u32,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_create_directory(
    mut _env: WasmEnvMut,
    _: u32,
    _: u32,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_remove_directory(
    mut _env: WasmEnvMut,
    _: u32,
    _: u32,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_readlink(
    mut _env: WasmEnvMut,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_rename(
    mut _env: WasmEnvMut,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_filestat_get(
    mut _env: WasmEnvMut,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn path_unlink_file(mut _env: WasmEnvMut, _: u32, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_prestat_get(mut _env: WasmEnvMut, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_prestat_dir_name(mut _env: WasmEnvMut, _: u32, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_filestat_get(mut _env: WasmEnvMut, _fd: u32, _filestat: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_filestat_set_size(mut _env: WasmEnvMut, _fd: u32, _: u64) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_pread(
    mut _env: WasmEnvMut,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn fd_pwrite(
    mut _env: WasmEnvMut,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn sock_accept(mut _env: WasmEnvMut, _fd: u32, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn sock_shutdown(mut _env: WasmEnvMut, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_BADF)
}

pub fn sched_yield(mut _env: WasmEnvMut) -> Result<Errno, Escape> {
    Ok(ERRNO_SUCCESS)
}

pub fn clock_time_get(
    mut env: WasmEnvMut,
    _clock_id: u32,
    _precision: u64,
    time: Uptr,
) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);
    caller_env.wenv.go_state.time += caller_env.wenv.go_state.time_interval;
    caller_env.caller_write_u32(time, caller_env.wenv.go_state.time as u32);
    caller_env.caller_write_u32(time + 4, (caller_env.wenv.go_state.time >> 32) as u32);
    Ok(ERRNO_SUCCESS)
}

pub fn random_get(mut env: WasmEnvMut, mut buf: u32, mut len: u32) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);

    while len >= 4 {
        let next_rand = caller_env.wenv.go_state.rng.next_u32();
        caller_env.caller_write_u32(buf, next_rand);
        buf += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = caller_env.wenv.go_state.rng.next_u32();
        for _ in 0..len {
            caller_env.caller_write_u8(buf, rem as u8);
            buf += 1;
            rem >>= 8;
        }
    }
    Ok(ERRNO_SUCCESS)
}

pub fn args_sizes_get(
    mut env: WasmEnvMut,
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);
    caller_env.caller_write_u32(length_ptr, 1);
    caller_env.caller_write_u32(data_size_ptr, 4);
    Ok(ERRNO_SUCCESS)
}

pub fn args_get(mut env: WasmEnvMut, argv_buf: Uptr, data_buf: Uptr) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);

    caller_env.caller_write_u32(argv_buf, data_buf as u32);
    caller_env.caller_write_u32(data_buf, 0x6E6962); // "bin\0"
    Ok(ERRNO_SUCCESS)
}

// we always simulate a timeout
pub fn poll_oneoff(
    mut env: WasmEnvMut,
    in_subs: Uptr,
    out_evt: Uptr,
    nsubscriptions: u32,
    nevents_ptr: Uptr,
) -> Result<Errno, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);

    const SUBSCRIPTION_SIZE: u32 = 48;
    for i in 0..nsubscriptions {
        let subs_base = in_subs + (SUBSCRIPTION_SIZE * (i as u32));
        let subs_type = caller_env.caller_read_u32(subs_base + 8);
        if subs_type != 0 {
            // not a clock subscription type
            continue;
        }
        let user_data = caller_env.caller_read_u32(subs_base);
        caller_env.caller_write_u32(out_evt, user_data);
        caller_env.caller_write_u32(out_evt + 8, 0);
        caller_env.caller_write_u32(nevents_ptr, 1);
        return Ok(ERRNO_SUCCESS);
    }
    Ok(ERRNO_INTVAL)
}

pub fn fd_fdstat_get(mut _env: WasmEnvMut, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_INTVAL)
}

pub fn fd_fdstat_set_flags(mut _env: WasmEnvMut, _: u32, _: u32) -> Result<Errno, Escape> {
    Ok(ERRNO_INTVAL)
}
