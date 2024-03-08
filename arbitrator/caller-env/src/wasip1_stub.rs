// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//! A stub impl of [WASI Preview 1][Wasi] for proving fraud.
//!
//! [Wasi]: https://github.com/WebAssembly/WASI/blob/main/legacy/preview1/docs.md

#![allow(clippy::too_many_arguments)]

use crate::{ExecEnv, GuestPtr, MemAccess};

#[repr(transparent)]
pub struct Errno(pub(crate) u16);

pub const ERRNO_SUCCESS: Errno = Errno(0);
pub const ERRNO_BADF: Errno = Errno(8);
pub const ERRNO_INVAL: Errno = Errno(28);

/// Writes the number and total size of args passed by the OS.
/// Note that this currently consists of just the program name `bin`.
pub fn args_sizes_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _: &mut E,
    length_ptr: GuestPtr,
    data_size_ptr: GuestPtr,
) -> Errno {
    mem.write_u32(length_ptr, 1);
    mem.write_u32(data_size_ptr, 4);
    ERRNO_SUCCESS
}

/// Writes the args passed by the OS.
/// Note that this currently consists of just the program name `bin`.
pub fn args_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _: &mut E,
    argv_buf: GuestPtr,
    data_buf: GuestPtr,
) -> Errno {
    mem.write_u32(argv_buf, data_buf.into());
    mem.write_u32(data_buf, 0x6E6962); // "bin\0"
    ERRNO_SUCCESS
}

/// Writes the number and total size of OS environment variables.
/// Note that none exist in Nitro.
pub fn environ_sizes_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    length_ptr: GuestPtr,
    data_size_ptr: GuestPtr,
) -> Errno {
    mem.write_u32(length_ptr, 0);
    mem.write_u32(data_size_ptr, 0);
    ERRNO_SUCCESS
}

/// Writes the number and total size of OS environment variables.
/// Note that none exist in Nitro.
pub fn environ_get<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: GuestPtr,
    _: GuestPtr,
) -> Errno {
    ERRNO_SUCCESS
}

/// Writes to the given file descriptor.
/// Note that we only support stdout and stderr.
/// Writing to output doesn't happen here.
/// in arbitrator that's in host_call_hook,
/// in jit it's in fd_write_wrapper
pub fn fd_write<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    fd: u32,
    iovecs_ptr: GuestPtr,
    iovecs_len: u32,
    ret_ptr: GuestPtr,
) -> Errno {
    if fd != 1 && fd != 2 {
        return ERRNO_BADF;
    }
    let mut size = 0;
    for i in 0..iovecs_len {
        let ptr = iovecs_ptr + i * 8;
        let len = mem.read_u32(ptr + 4);
        size += len;
    }
    mem.write_u32(ret_ptr, size);
    ERRNO_SUCCESS
}

/// Closes the given file descriptor. Unsupported.
pub fn fd_close<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32) -> Errno {
    ERRNO_BADF
}

/// Reads from the given file descriptor. Unsupported.
pub fn fd_read<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Reads the contents of a directory. Unsupported.
pub fn fd_readdir<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Syncs a file to disk. Unsupported.
pub fn fd_sync<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32) -> Errno {
    ERRNO_SUCCESS
}

/// Move within a file. Unsupported.
pub fn fd_seek<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _offset: u64,
    _whence: u8,
    _filesize: u32,
) -> Errno {
    ERRNO_BADF
}

/// Syncs file contents to disk. Unsupported.
pub fn fd_datasync<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _fd: u32) -> Errno {
    ERRNO_BADF
}

/// Retrieves attributes about a file descriptor. Unsupported.
pub fn fd_fdstat_get<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_INVAL
}

/// Sets the attributes of a file descriptor. Unsupported.
pub fn fd_fdstat_set_flags<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_INVAL
}

/// Opens the file or directory at the given path. Unsupported.
pub fn path_open<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
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
pub fn path_create_directory<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Unlinks a directory. Unsupported.
pub fn path_remove_directory<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Resolves a symbolic link. Unsupported.
pub fn path_readlink<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
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
pub fn path_rename<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
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
pub fn path_filestat_get<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Unlinks the file at the given path. Unsupported.
pub fn path_unlink_file<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about a file. Unsupported.
pub fn fd_prestat_get<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about a directory. Unsupported.
pub fn fd_prestat_dir_name<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Retrieves info about a file. Unsupported.
pub fn fd_filestat_get<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _filestat: u32,
) -> Errno {
    ERRNO_BADF
}

/// Sets the size of an open file. Unsupported.
pub fn fd_filestat_set_size<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u64,
) -> Errno {
    ERRNO_BADF
}

/// Peaks within a descriptor without modifying its state. Unsupported.
pub fn fd_pread<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Writes to a descriptor without modifying the current offset. Unsupported.
pub fn fd_pwrite<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Accepts a new connection. Unsupported.
pub fn sock_accept<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

/// Shuts down a socket. Unsupported.
pub fn sock_shutdown<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

/// Yields execution to the OS scheduler. Effectively does nothing in Nitro due to the lack of threads.
pub fn sched_yield<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E) -> Errno {
    ERRNO_SUCCESS
}

/// 10ms in ns
static TIME_INTERVAL: u64 = 10_000_000;

/// Retrieves the time in ns of the given clock.
/// Note that in Nitro, all clocks point to the same deterministic counter that advances 10ms whenever
/// this function is called.
pub fn clock_time_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    env: &mut E,
    _clock_id: u32,
    _precision: u64,
    time_ptr: GuestPtr,
) -> Errno {
    env.advance_time(TIME_INTERVAL);
    mem.write_u64(time_ptr, env.get_time());
    ERRNO_SUCCESS
}

/// Fills a slice with psuedo-random bytes.
/// Note that in Nitro, the bytes are deterministically generated from a common seed.
pub fn random_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    env: &mut E,
    mut buf: GuestPtr,
    mut len: u32,
) -> Errno {
    while len >= 4 {
        let next_rand = env.next_rand_u32();
        mem.write_u32(buf, next_rand);
        buf += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = env.next_rand_u32();
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
pub fn poll_oneoff<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    env: &mut E,
    in_subs: GuestPtr,
    out_evt: GuestPtr,
    num_subscriptions: u32,
    num_events_ptr: GuestPtr,
) -> Errno {
    // simulate the passage of time each poll request
    env.advance_time(TIME_INTERVAL);

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
