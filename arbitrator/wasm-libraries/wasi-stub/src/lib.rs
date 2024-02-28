// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_std]

use rand::RngCore;
use paste::paste;
use callerenv::{
    self,
    wasip1_stub::{Errno, Uptr},
    CallerEnv,
};
use rand_pcg::Pcg32;

#[allow(dead_code)]
extern "C" {
    fn wavm_caller_load8(ptr: Uptr) -> u8;
    fn wavm_caller_load32(ptr: Uptr) -> u32;
    fn wavm_caller_store8(ptr: Uptr, val: u8);
    fn wavm_caller_store32(ptr: Uptr, val: u32);
    fn wavm_halt_and_set_finished() -> !;
}

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

static mut TIME: u64 = 0;
static mut RNG: Option<Pcg32> = None;

#[derive(Default)]
struct StaticCallerEnv{}

impl CallerEnv<'static> for StaticCallerEnv {
    fn read_u8(&self, ptr: u32) -> u8 {
        unsafe {
            wavm_caller_load8(ptr)
        }
    }

    fn read_u16(&self, ptr: u32) -> u16 {
        let lsb = self.read_u8(ptr);
        let msb = self.read_u8(ptr+1);
        (msb as u16) << 8 | (lsb as u16)
    }

    fn read_u32(&self, ptr: u32) -> u32 {
        let lsb = self.read_u16(ptr);
        let msb = self.read_u16(ptr+2);
        (msb as u32) << 16 | (lsb as u32)
    }

    fn read_u64(&self, ptr: u32) -> u64 {
        let lsb = self.read_u32(ptr);
        let msb = self.read_u32(ptr+4);
        (msb as u64) << 32 | (lsb as u64)
    }

    fn write_u8(&mut self, ptr: u32, x: u8) -> &mut Self {
        unsafe {
            wavm_caller_store8(ptr, x);
        }
        self
    }

    fn write_u16(&mut self, ptr: u32, x: u16) -> &mut Self {
        self.write_u8(ptr, (x & 0xff) as u8);
        self.write_u8(ptr + 1, ((x >> 8) & 0xff) as u8);
        self
    }

    fn write_u32(&mut self, ptr: u32, x: u32) -> &mut Self {
        self.write_u16(ptr, (x & 0xffff) as u16);
        self.write_u16(ptr + 2, ((x >> 16) & 0xffff) as u16);
        self
    }

    fn write_u64(&mut self, ptr: u32, x: u64) -> &mut Self {
        self.write_u32(ptr, (x & 0xffffffff) as u32);
        self.write_u32(ptr + 4, ((x >> 16) & 0xffffffff) as u32);
        self
    }

    fn print_string(&mut self, _ptr: u32, _len: u32) {} // TODO?

    fn get_time(&self) -> u64 {
        unsafe {
            TIME
        }
    }

    fn advance_time(&mut self, delta: u64) {
        unsafe {
            TIME += delta
        }
    }

    fn next_rand_u32(&mut self) -> u32 {
        unsafe {
            RNG.get_or_insert_with(|| callerenv::create_pcg())
        }
        .next_u32()
    }
}


macro_rules! wrap {
    ($func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty) => {
        paste! {
            #[no_mangle]
            pub unsafe extern "C" fn [<wasi_snapshot_preview1__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                let caller_env = StaticCallerEnv::default();

                callerenv::wasip1_stub::$func_name(caller_env, $($arg_name),*)
            }
        }
    };
}

wrap!(clock_time_get(
    clock_id: u32,
    precision: u64,
    time_ptr: Uptr
) -> Errno);

wrap!(random_get(buf: Uptr, len: u32) -> Errno);

wrap!(environ_sizes_get(length_ptr: Uptr, data_size_ptr: Uptr) -> Errno);
wrap!(fd_write(
    fd: u32,
    iovecs_ptr: Uptr,
    iovecs_len: u32,
    ret_ptr: Uptr
) -> Errno);
wrap!(environ_get(a: u32, b: u32) -> Errno);
wrap!(fd_close(fd: u32) -> Errno);
wrap!(fd_read(a: u32, b: u32, c: u32, d: u32) -> Errno);
wrap!(fd_readdir(
    fd: u32,
    a: u32,
    b: u32,
    c: u64,
    d: u32
) -> Errno);

wrap!(fd_sync(a: u32) -> Errno);

wrap!(fd_seek(
    _fd: u32,
    _offset: u64,
    _whence: u8,
    _filesize: u32
) -> Errno);

wrap!(fd_datasync(_fd: u32) -> Errno);

wrap!(path_open(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32,
    f: u64,
    g: u64,
    h: u32,
    i: u32
) -> Errno);

wrap!(path_create_directory(
    a: u32,
    b: u32,
    c: u32
) -> Errno);

wrap!(path_remove_directory(
    a: u32,
    b: u32,
    c: u32
) -> Errno);

wrap!(path_readlink(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32,
    f: u32
) -> Errno);

wrap!(path_rename(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32,
    f: u32
) -> Errno);

wrap!(path_filestat_get(
    a: u32,
    b: u32,
    c: u32,
    d: u32,
    e: u32
) -> Errno);

wrap!(path_unlink_file(a: u32, b: u32, c: u32) -> Errno);

wrap!(fd_prestat_get(a: u32, b: u32) -> Errno);

wrap!(fd_prestat_dir_name(a: u32, b: u32, c: u32) -> Errno);

wrap!(fd_filestat_get(_fd: u32, _filestat: u32) -> Errno);

wrap!(fd_filestat_set_size(_fd: u32, size: u64) -> Errno);

wrap!(fd_pread(
    _fd: u32,
    _a: u32,
    _b: u32,
    _c: u64,
    _d: u32
) -> Errno);

wrap!(fd_pwrite(
    _fd: u32,
    _a: u32,
    _b: u32,
    _c: u64,
    _d: u32
) -> Errno);

wrap!(sock_accept(_fd: u32, a: u32, b: u32) -> Errno);

wrap!(sock_shutdown(a: u32, b: u32) -> Errno);

wrap!(sched_yield() -> Errno);

wrap!(args_sizes_get(
    length_ptr: Uptr,
    data_size_ptr: Uptr
) -> Errno);

wrap!(args_get(argv_buf: Uptr, data_buf: Uptr) -> Errno);

wrap!(poll_oneoff(
    in_subs: Uptr,
    out_evt: Uptr,
    nsubscriptions: u32,
    nevents_ptr: Uptr
) -> Errno);

wrap!(fd_fdstat_get(a: u32, b: u32) -> Errno);

wrap!(fd_fdstat_set_flags(a: u32, b: u32) -> Errno);
