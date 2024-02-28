use crate::CallerEnv;

pub type Errno = u16;

pub type Uptr = u32;

pub const ERRNO_SUCCESS: Errno = 0;
pub const ERRNO_BADF: Errno = 8;
pub const ERRNO_INTVAL: Errno = 28;

pub fn environ_sizes_get<'a, E: CallerEnv<'a>>(
    caller_env: &mut E,
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Errno {
    caller_env.write_u32(length_ptr, 0);
    caller_env.write_u32(data_size_ptr, 0);
    ERRNO_SUCCESS
}

pub fn fd_write<'a, E: CallerEnv<'a>>(
    caller_env: &mut E,
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
        let iovec = caller_env.read_u32(ptr);
        let len = caller_env.read_u32(ptr + 4);
        caller_env.print_string(iovec, len);
        size += len;
    }
    caller_env.write_u32(ret_ptr, size);
    ERRNO_SUCCESS
}

pub fn environ_get<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_INTVAL
}

pub fn fd_close<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_read<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_readdir<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_sync<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32) -> Errno {
    ERRNO_SUCCESS
}

pub fn fd_seek<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _fd: u32,
    _offset: u64,
    _whence: u8,
    _filesize: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_datasync<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _fd: u32) -> Errno {
    ERRNO_BADF
}

pub fn path_open<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
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

pub fn path_create_directory<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_remove_directory<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_readlink<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_rename<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_filestat_get<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_unlink_file<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_prestat_get<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_prestat_dir_name<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_filestat_get<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _fd: u32,
    _filestat: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_filestat_set_size<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _fd: u32, _: u64) -> Errno {
    ERRNO_BADF
}

pub fn fd_pread<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_pwrite<'a, E: CallerEnv<'a>>(
    _caller_env: &mut E,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn sock_accept<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _fd: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn sock_shutdown<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn sched_yield<'a, E: CallerEnv<'a>>(_caller_env: &mut E) -> Errno {
    ERRNO_SUCCESS
}

static TIME_INTERVAL: u64 = 10_000_000;

pub fn clock_time_get<'a, E: CallerEnv<'a>>(
    caller_env: &mut E,
    _clock_id: u32,
    _precision: u64,
    time_ptr: Uptr,
) -> Errno {
    caller_env.advance_time(TIME_INTERVAL);
    caller_env.write_u64(time_ptr, caller_env.get_time());
    ERRNO_SUCCESS
}

pub fn random_get<'a, E: CallerEnv<'a>>(caller_env: &mut E, mut buf: u32, mut len: u32) -> Errno {
    while len >= 4 {
        let next_rand = caller_env.next_rand_u32();
        caller_env.write_u32(buf, next_rand);
        buf += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = caller_env.next_rand_u32();
        for _ in 0..len {
            caller_env.write_u8(buf, rem as u8);
            buf += 1;
            rem >>= 8;
        }
    }
    ERRNO_SUCCESS
}

pub fn args_sizes_get<'a, E: CallerEnv<'a>>(
    caller_env: &mut E,
    length_ptr: Uptr,
    data_size_ptr: Uptr,
) -> Errno {
    caller_env.write_u32(length_ptr, 1);
    caller_env.write_u32(data_size_ptr, 4);
    ERRNO_SUCCESS
}

pub fn args_get<'a, E: CallerEnv<'a>>(caller_env: &mut E, argv_buf: Uptr, data_buf: Uptr) -> Errno {
    caller_env.write_u32(argv_buf, data_buf as u32);
    caller_env.write_u32(data_buf, 0x6E6962); // "bin\0"
    ERRNO_SUCCESS
}

// we always simulate a timeout
pub fn poll_oneoff<'a, E: CallerEnv<'a>>(
    caller_env: &mut E,
    in_subs: Uptr,
    out_evt: Uptr,
    nsubscriptions: u32,
    nevents_ptr: Uptr,
) -> Errno {
    const SUBSCRIPTION_SIZE: u32 = 48;
    for i in 0..nsubscriptions {
        let subs_base = in_subs + (SUBSCRIPTION_SIZE * (i as u32));
        let subs_type = caller_env.read_u32(subs_base + 8);
        if subs_type != 0 {
            // not a clock subscription type
            continue;
        }
        let user_data = caller_env.read_u32(subs_base);
        caller_env.write_u32(out_evt, user_data);
        caller_env.write_u32(out_evt + 8, 0);
        caller_env.write_u32(nevents_ptr, 1);
        return ERRNO_SUCCESS;
    }
    ERRNO_INTVAL
}

pub fn fd_fdstat_get<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_INTVAL
}

pub fn fd_fdstat_set_flags<'a, E: CallerEnv<'a>>(_caller_env: &mut E, _: u32, _: u32) -> Errno {
    ERRNO_INTVAL
}
