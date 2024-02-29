use crate::{ExecEnv, MemAccess};

pub type Errno = u16;

pub use crate::Uptr;

pub const ERRNO_SUCCESS: Errno = 0;
pub const ERRNO_BADF: Errno = 8;
pub const ERRNO_INTVAL: Errno = 28;

pub fn environ_sizes_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Errno {
    mem.write_u32(length_ptr, 0);
    mem.write_u32(data_size_ptr, 0);
    ERRNO_SUCCESS
}

pub fn fd_write<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    env: &mut E,
    fd: u32,
    iovecs_ptr: Uptr,
    iovecs_len: u32,
    ret_ptr: Uptr,
) -> Errno {
    if fd != 1 && fd != 2 {
        return ERRNO_BADF;
    }
    let mut size = 0;
    for i in 0..iovecs_len {
        let ptr = iovecs_ptr + i * 8;
        // let iovec = mem.read_u32(ptr);
        let len = mem.read_u32(ptr + 4);
        let data = mem.read_slice(ptr, len as usize);
        env.print_string(&data);
        size += len;
    }
    mem.write_u32(ret_ptr, size);
    ERRNO_SUCCESS
}

pub fn environ_get<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_INTVAL
}

pub fn fd_close<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32) -> Errno {
    ERRNO_BADF
}

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

pub fn fd_sync<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32) -> Errno {
    ERRNO_SUCCESS
}

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

pub fn fd_datasync<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _fd: u32) -> Errno {
    ERRNO_BADF
}

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

pub fn path_create_directory<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_remove_directory<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

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

pub fn path_unlink_file<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_prestat_get<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_prestat_dir_name<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_filestat_get<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _filestat: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_filestat_set_size<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u64,
) -> Errno {
    ERRNO_BADF
}

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

pub fn sock_accept<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn sock_shutdown<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn sched_yield<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E) -> Errno {
    ERRNO_SUCCESS
}

static TIME_INTERVAL: u64 = 10_000_000;

pub fn clock_time_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    env: &mut E,
    _clock_id: u32,
    _precision: u64,
    time_ptr: Uptr,
) -> Errno {
    env.advance_time(TIME_INTERVAL);
    mem.write_u64(time_ptr, env.get_time());
    ERRNO_SUCCESS
}

pub fn random_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    env: &mut E,
    mut buf: u32,
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

pub fn args_sizes_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _: &mut E,
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Errno {
    mem.write_u32(length_ptr, 1);
    mem.write_u32(data_size_ptr, 4);
    ERRNO_SUCCESS
}

pub fn args_get<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _: &mut E,
    argv_buf: Uptr,
    data_buf: Uptr,
) -> Errno {
    mem.write_u32(argv_buf, data_buf as u32);
    mem.write_u32(data_buf, 0x6E6962); // "bin\0"
    ERRNO_SUCCESS
}

// we always simulate a timeout
pub fn poll_oneoff<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _: &mut E,
    in_subs: Uptr,
    out_evt: Uptr,
    nsubscriptions: u32,
    nevents_ptr: Uptr,
) -> Errno {
    const SUBSCRIPTION_SIZE: u32 = 48;
    for i in 0..nsubscriptions {
        let subs_base = in_subs + (SUBSCRIPTION_SIZE * (i as u32));
        let subs_type = mem.read_u32(subs_base + 8);
        if subs_type != 0 {
            // not a clock subscription type
            continue;
        }
        let user_data = mem.read_u32(subs_base);
        mem.write_u32(out_evt, user_data);
        mem.write_u32(out_evt + 8, 0);
        mem.write_u32(nevents_ptr, 1);
        return ERRNO_SUCCESS;
    }
    ERRNO_INTVAL
}

pub fn fd_fdstat_get<M: MemAccess, E: ExecEnv>(_: &mut M, _: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_INTVAL
}

pub fn fd_fdstat_set_flags<M: MemAccess, E: ExecEnv>(
    _: &mut M,
    _: &mut E,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_INTVAL
}
