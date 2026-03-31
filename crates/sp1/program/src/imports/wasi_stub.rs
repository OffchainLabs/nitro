//! WASI stubs — thin wrappers delegating to caller_env::wasip1_stub.

use wasmer::FunctionEnvMut;

use crate::{
    Ptr, platform,
    replay::CustomEnvData,
    state::{gp, sp1_env},
};

pub fn proc_exit(mut ctx: FunctionEnvMut<CustomEnvData>, code: u32) {
    let (data, _store) = ctx.data_and_store_mut();

    if code == 0 {
        platform::print_string(
            1,
            format!(
                "Validation succeeds with hash 0x{}",
                hex::encode(data.input().large_globals[0])
            )
            .as_bytes(),
        );
    }

    platform::exit(code);
}

macro_rules! wrap {
    (fn $name:ident($($arg:ident: $ty:tt),* $(,)?)) => {
        pub fn $name(mut src: FunctionEnvMut<CustomEnvData>, $($arg: $ty),*) -> u16 {
            let (mut mem, state) = sp1_env(&mut src);
            caller_env::wasip1_stub::$name(&mut mem, state, $(wrap!(@conv $arg $ty)),*).0
        }
    };
    (@conv $arg:ident Ptr) => { gp($arg) };
    (@conv $arg:ident $ty:tt) => { $arg };
}

wrap!(fn clock_time_get(_clock_id: u32, _precision: u64, time_ptr: Ptr));
wrap!(fn random_get(buf: Ptr, len: u32));
wrap!(fn environ_get(a: Ptr, b: Ptr));
wrap!(fn environ_sizes_get(length_ptr: Ptr, data_size_ptr: Ptr));
wrap!(fn fd_read(a: u32, b: u32, c: u32, d: u32));
wrap!(fn fd_close(fd: u32));
wrap!(fn fd_write(fd: u32, iovecs_ptr: Ptr, iovecs_len: u32, ret_ptr: Ptr));
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
wrap!(fn args_sizes_get(length_ptr: Ptr, data_size_ptr: Ptr));
wrap!(fn args_get(argv_buf: Ptr, data_buf: Ptr));
wrap!(fn fd_fdstat_get(a: u32, b: u32));
wrap!(fn fd_fdstat_set_flags(a: u32, b: u32));
wrap!(fn poll_oneoff(in_subs: Ptr, out_evt: Ptr, nsubscriptions: u32, nevents_ptr: Ptr));
