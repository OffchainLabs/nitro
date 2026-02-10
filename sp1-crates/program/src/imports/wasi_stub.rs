//! WASI stubs
//!
//! The code used here is heavily borrowed from nitro's own WASI stubs:
//! https://github.com/OffchainLabs/nitro/blob/c858ed93a5a4fd81908277d94fb72974058a3615/arbitrator/caller-env/src/wasip1_stub.rs

use crate::{Escape, Ptr, platform, read_slice, replay::CustomEnvData};
use rand::RngCore;
use wasmer::FunctionEnvMut;

pub type Errno = u16;

pub const ERRNO_SUCCESS: Errno = 0;
pub const ERRNO_BADF: Errno = 8;
pub const ERRNO_INVAL: Errno = 28;

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
) -> Result<Errno, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    argc.write(&memory, 1)?;
    argv_buf_size.write(&memory, 4)?;

    Ok(ERRNO_SUCCESS)
}

pub fn args_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    argv_buf: Ptr,
    data_buf: Ptr,
) -> Result<Errno, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let data_buf = data_buf.deref(&memory);

    argv_buf.write(&memory, data_buf.offset() as u32)?;
    data_buf.write(0x6E6962)?; // "bin\0"

    Ok(ERRNO_SUCCESS)
}

pub fn environ_sizes_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    length_ptr: Ptr,
    data_size_ptr: Ptr,
) -> Result<Errno, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    length_ptr.write(&memory, 0)?;
    data_size_ptr.write(&memory, 0)?;

    Ok(ERRNO_SUCCESS)
}

pub fn environ_get(_ctx: FunctionEnvMut<CustomEnvData>, _: Ptr, _: Ptr) -> Errno {
    ERRNO_SUCCESS
}

pub fn fd_write(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    fd: u32,
    iovecs_ptr: Ptr,
    iovecs_len: u32,
    ret_ptr: Ptr,
) -> Result<Errno, Escape> {
    if fd != 1 && fd != 2 {
        return Ok(ERRNO_BADF);
    }

    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let mut size = 0;
    for i in 0..iovecs_len {
        let ptr = iovecs_ptr.add_offset((i * 2).into()).unwrap();
        let len = ptr.add_offset(1).unwrap().read(&memory)?;
        let ptr = Ptr::new(ptr.read(&memory)?);
        let data = read_slice(ptr, len as usize, &memory)?;

        platform::print_string(fd, &data);

        size += len;
    }

    ret_ptr.write(&memory, size)?;
    Ok(ERRNO_SUCCESS)
}

pub fn fd_close(_ctx: FunctionEnvMut<CustomEnvData>, _fd: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_read(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_readdir(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_sync(_ctx: FunctionEnvMut<CustomEnvData>, _: u32) -> Errno {
    ERRNO_SUCCESS
}

pub fn fd_seek(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u64, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_datasync(_ctx: FunctionEnvMut<CustomEnvData>, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_prestat_get(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_prestat_dir_name(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_filestat_get(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn fd_filestat_set_size(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u64) -> Errno {
    ERRNO_BADF
}

pub fn fd_pread(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_pwrite(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _fd: u32,
    _: u32,
    _: u32,
    _: u64,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn fd_fdstat_get(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32) -> Errno {
    ERRNO_INVAL
}

pub fn fd_fdstat_set_flags(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32) -> Errno {
    ERRNO_INVAL
}

const TIME_INTERVAL: u64 = 10_000_000;

pub fn clock_time_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    _clock_id: u32,
    _precision: u64,
    time_ptr: Ptr,
) -> Result<Errno, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.time += TIME_INTERVAL;
    time_ptr.cast::<u64>().write(&memory, data.time)?;

    Ok(ERRNO_SUCCESS)
}

pub fn random_get(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    mut buf: Ptr,
    mut len: u32,
) -> Result<Errno, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    while len >= 4 {
        let next_rand = data.pcg.next_u32();
        buf.write(&memory, next_rand)?;
        buf = buf.add_offset(1).unwrap();
        len -= 4;
    }
    if len > 0 {
        let mut rem = data.pcg.next_u32();
        let mut buf = buf.cast::<u8>();

        for _ in 0..len {
            buf.write(&memory, rem as u8)?;
            buf = buf.add_offset(1).unwrap();
            rem >>= 8;
        }
    }
    Ok(ERRNO_SUCCESS)
}

pub fn poll_oneoff(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_subs: Ptr,
    out_evt: Ptr,
    num_subscriptions: u32,
    num_events_ptr: Ptr,
) -> Result<Errno, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.time += TIME_INTERVAL;

    const SUBSCRIPTION_SIZE: u32 = 48;
    for index in 0..num_subscriptions {
        let subs_base = in_subs
            .cast::<u8>()
            .add_offset(SUBSCRIPTION_SIZE * index)
            .unwrap();
        let subs_type = subs_base
            .add_offset(8)
            .unwrap()
            .cast::<u32>()
            .read(&memory)?;
        if subs_type != 0 {
            continue;
        }
        let user_data = subs_base.cast::<u32>().read(&memory)?;
        out_evt.write(&memory, user_data)?;
        out_evt.add_offset(2).unwrap().write(&memory, subs_type)?;
        num_events_ptr.write(&memory, 1)?;
        return Ok(ERRNO_SUCCESS);
    }
    Ok(ERRNO_INVAL)
}

pub fn sched_yield(_ctx: FunctionEnvMut<CustomEnvData>) -> Errno {
    ERRNO_SUCCESS
}

pub fn path_open(
    _ctx: FunctionEnvMut<CustomEnvData>,
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

pub fn path_create_directory(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn path_remove_directory(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn path_readlink(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_rename(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_filestat_get(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
    _: u32,
) -> Errno {
    ERRNO_BADF
}

pub fn path_unlink_file(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn sock_accept(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}

pub fn sock_shutdown(_ctx: FunctionEnvMut<CustomEnvData>, _: u32, _: u32) -> Errno {
    ERRNO_BADF
}
