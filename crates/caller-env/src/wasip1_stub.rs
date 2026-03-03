// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! A stub impl of [WASI Preview 1][Wasi] for proving fraud.
//!
//! [Wasi]: https://github.com/WebAssembly/WASI/blob/main/legacy/preview1/docs.md

#![allow(clippy::too_many_arguments)]

use crate::{ExecEnv, GuestPtr, MemAccess, wavmio::WavmEnv};

#[repr(transparent)]
pub struct Errno(pub u16);

pub const ERRNO_SUCCESS: Errno = Errno(0);
pub const ERRNO_BADF: Errno = Errno(8);
pub const ERRNO_INVAL: Errno = Errno(28);

/// Writes the number and total size of args passed by the OS.
/// Note that this currently consists of just the program name `bin`.
pub fn args_sizes_get(
    env: &mut impl WavmEnv,
    length_ptr: GuestPtr,
    data_size_ptr: GuestPtr,
) -> Errno {
    let (mut mem, _) = env.wavm_env();
    mem.write_u32(length_ptr, 1);
    mem.write_u32(data_size_ptr, 4);
    ERRNO_SUCCESS
}

/// Writes the args passed by the OS.
/// Note that this currently consists of just the program name `bin`.
pub fn args_get(
    env: &mut impl WavmEnv,
    argv_buf: GuestPtr,
    data_buf: GuestPtr,
) -> Errno {
    let (mut mem, _) = env.wavm_env();
    mem.write_u32(argv_buf, data_buf.into());
    mem.write_u32(data_buf, 0x6E6962); // "bin\0"
    ERRNO_SUCCESS
}

/// Writes the number and total size of OS environment variables.
/// Note that none exist in Nitro.
pub fn environ_sizes_get(
    env: &mut impl WavmEnv,
    length_ptr: GuestPtr,
    data_size_ptr: GuestPtr,
) -> Errno {
    let (mut mem, _) = env.wavm_env();
    mem.write_u32(length_ptr, 0);
    mem.write_u32(data_size_ptr, 0);
    ERRNO_SUCCESS
}

/// Writes the number and total size of OS environment variables.
/// Note that none exist in Nitro.
pub fn environ_get(
    _: &mut impl WavmEnv,
    _: GuestPtr,
    _: GuestPtr,
) -> Errno {
    ERRNO_SUCCESS
}

/// Writes to the given file descriptor.
/// Note that we only support stdout and stderr.
pub fn fd_write(
    env: &mut impl WavmEnv,
    fd: u32,
    iovecs_ptr: GuestPtr,
    iovecs_len: u32,
    ret_ptr: GuestPtr,
) -> Errno {
    if fd != 1 && fd != 2 {
        return ERRNO_BADF;
    }
    let (mut mem, mut state) = env.wavm_env();
    let mut size = 0;
    for i in 0..iovecs_len {
        let ptr = iovecs_ptr + i * 8;
        let len = mem.read_u32(ptr + 4);
        let ptr = mem.read_u32(ptr); // TODO: string might be split across utf-8 character boundary
        let data = mem.read_slice(GuestPtr(ptr), len as usize);
        state.print_string(&data);
        size += len;
    }
    mem.write_u32(ret_ptr, size);
    ERRNO_SUCCESS
}

/// Closes the given file descriptor. Unsupported.
pub fn fd_close(_: &mut impl WavmEnv, _: u32) -> Errno {
    ERRNO_BADF
}

/// Reads from the given file descriptor. Unsupported.
pub fn fd_read(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Reads the contents of a directory. Unsupported.
pub fn fd_readdir(
    _: &mut impl WavmEnv,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Syncs a file to disk. Unsupported.
pub fn fd_sync(_: &mut impl WavmEnv, _: u32) -> Errno {
    ERRNO_SUCCESS
}

/// Move within a file. Unsupported.
pub fn fd_seek(
    _: &mut impl WavmEnv,
    _fd: u32,
    _offset: u64,
    _whence: u8,
    _filesize: u32,
) -> Errno {
    ERRNO_BADF
}

/// Syncs file contents to disk. Unsupported.
pub fn fd_datasync(_: &mut impl WavmEnv, _fd: u32) -> Errno {
    ERRNO_BADF
}

/// Retrieves attributes about a file descriptor. Unsupported.
pub fn fd_fdstat_get(_: &mut impl WavmEnv, _: u32, _: u32) -> Errno {
    ERRNO_INVAL
}

/// Sets the attributes of a file descriptor. Unsupported.
pub fn fd_fdstat_set_flags(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_INVAL
}

/// Opens the file or directory at the given path. Unsupported.
pub fn path_open(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u64,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Creates a directory. Unsupported.
pub fn path_create_directory(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Unlinks a directory. Unsupported.
pub fn path_remove_directory(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Resolves a symbolic link. Unsupported.
pub fn path_readlink(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Moves a file. Unsupported.
pub fn path_rename(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about an open file. Unsupported.
pub fn path_filestat_get(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Unlinks the file at the given path. Unsupported.
pub fn path_unlink_file(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about a file. Unsupported.
pub fn fd_prestat_get(_: &mut impl WavmEnv, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about a directory. Unsupported.
pub fn fd_prestat_dir_name(
    _: &mut impl WavmEnv,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about a file. Unsupported.
pub fn fd_filestat_get(
    _: &mut impl WavmEnv,
    _fd: u32,
    _filestat: u32,
) -> Errno {
    ERRNO_BADF
}

/// Sets the size of an open file. Unsupported.
pub fn fd_filestat_set_size(
    _: &mut impl WavmEnv,
    _fd: u32,
    _: u64,
) -> Errno {
    ERRNO_BADF
}

/// Peaks within a descriptor without modifying its state. Unsupported.
pub fn fd_pread(
    _: &mut impl WavmEnv,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Writes to a descriptor without modifying the current offset. Unsupported.
pub fn fd_pwrite(
    _: &mut impl WavmEnv,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Accepts a new connection. Unsupported.
pub fn sock_accept(
    _: &mut impl WavmEnv,
    _fd: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Shuts down a socket. Unsupported.
pub fn sock_shutdown(_: &mut impl WavmEnv, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

/// Yields execution to the OS scheduler. Effectively does nothing in Nitro due to the lack of threads.
pub fn sched_yield(_: &mut impl WavmEnv) -> Errno {
    ERRNO_SUCCESS
}

/// 10ms in ns
static TIME_INTERVAL: u64 = 10_000_000;

/// Retrieves the time in ns of the given clock.
/// Note that in Nitro, all clocks point to the same deterministic counter that advances 10ms whenever
/// this function is called.
pub fn clock_time_get(
    env: &mut impl WavmEnv,
    _clock_id: u32,
    _precision: u64,
    time_ptr: GuestPtr,
) -> Errno {
    let (mut mem, mut state) = env.wavm_env();
    state.advance_time(TIME_INTERVAL);
    mem.write_u64(time_ptr, state.get_time());
    ERRNO_SUCCESS
}

/// Fills a slice with psuedo-random bytes.
/// Note that in Nitro, the bytes are deterministically generated from a common seed.
pub fn random_get(
    env: &mut impl WavmEnv,
    mut buf: GuestPtr,
    mut len: u32,
) -> Errno {
    let (mut mem, mut state) = env.wavm_env();
    while len >= 4 {
        let next_rand = state.next_rand_u32();
        mem.write_u32(buf, next_rand);
        buf += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = state.next_rand_u32();
        for _ in 0..len {
            mem.write_u8(buf, rem as u8);
            buf += 1;
            rem >>= 8;
        }
    }
    ERRNO_SUCCESS
}

/// Poll for events.
/// Note that we always simulate a timeout and skip all others.
pub fn poll_oneoff(
    env: &mut impl WavmEnv,
    in_subs: GuestPtr,
    out_evt: GuestPtr,
    num_subscriptions: u32,
    num_events_ptr: GuestPtr,
) -> Errno {
    let (mut mem, mut state) = env.wavm_env();
    // simulate the passage of time each poll request
    state.advance_time(TIME_INTERVAL);

    const SUBSCRIPTION_SIZE: u32 = 48; // user data + 40-byte union
    for index in 0..num_subscriptions {
        let subs_base = in_subs + (SUBSCRIPTION_SIZE * index);
        let subs_type = mem.read_u32(subs_base + 8);
        if subs_type != 0 {
            // not a clock subscription type
            continue;
        }
        let user_data = mem.read_u32(subs_base);
        mem.write_u32(out_evt, user_data);
        mem.write_u32(out_evt + 8, subs_type);
        mem.write_u32(num_events_ptr, 1);
        return ERRNO_SUCCESS;
    }
    ERRNO_INVAL
}
